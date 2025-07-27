package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"

	"emailchecker"
	"emailchecker/analyzer"
	"emailchecker/api"
	"emailchecker/disposable"
	"emailchecker/dns"
	"emailchecker/edu"
	"emailchecker/emailpattern"
	"emailchecker/sqlite"
	"emailchecker/wellknown"
)

func main() {
	app := &cli.App{
		Name:  "emailchecker",
		Usage: "Email validation and analysis tool",
		Commands: []*cli.Command{
			{
				Name:    "check",
				Aliases: []string{"c"},
				Usage:   "Check email(s) from various sources",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Usage:   "Read emails from file (one per line)",
					},
					&cli.BoolFlag{
						Name:    "stdin",
						Aliases: []string{"s"},
						Usage:   "Read emails from stdin (one per line)",
					},
				},
				Action: checkEmails,
			},
			{
				Name:    "server",
				Aliases: []string{"srv"},
				Usage:   "Start HTTP API server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Value:   "8080",
						Usage:   "Port to run the server on",
					},
				},
				Action: startServer,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createChecker() (*emailchecker.EmailChecker, error) {
	repo, err := sqlite.New("checker.db")
	if err != nil {
		return nil, err
	}

	netClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	disposableFetcher := disposable.NewGithubFetcher(netClient)
	dnsChecker := dns.New(netClient)
	dnsResolver := dns.NewResolver(dnsChecker, repo)

	disposableSvc, err := disposable.New(repo, disposableFetcher)
	if err != nil {
		return nil, err
	}

	analyzerSvc := analyzer.New()
	wellKnownFetcher := wellknown.NewTranco(netClient)

	welknownSvc, err := wellknown.New(repo, wellKnownFetcher)
	if err != nil {
		return nil, err
	}

	eduFetcher := edu.NewEduFetcher(netClient)
	eduChecker, err := edu.New(repo, eduFetcher)
	if err != nil {
		return nil, err
	}

	cfg := emailchecker.Config{
		DisposableService:        disposableSvc,
		DNSService:               dnsResolver,
		AnalysisService:          analyzerSvc,
		EmailPatternService:      emailpattern.New(),
		WellKnownService:         welknownSvc,
		EducationalDomainService: eduChecker,
	}

	return emailchecker.New(&cfg)
}

func checkEmails(c *cli.Context) error {
	checker, err := createChecker()
	if err != nil {
		return fmt.Errorf("failed to create checker: %v", err)
	}
	defer checker.Close()

	var emails []string

	if c.String("file") != "" {
		emails, err = readEmailsFromFile(c.String("file"))
		if err != nil {
			return fmt.Errorf("failed to read emails from file: %v", err)
		}
	} else if c.Bool("stdin") {
		emails, err = readEmailsFromStdin()
		if err != nil {
			return fmt.Errorf("failed to read emails from stdin: %v", err)
		}
	} else if c.NArg() > 0 {
		emails = c.Args().Slice()
	} else {
		return fmt.Errorf("please provide an email via argument, --file, or --stdin")
	}

	ctx := context.Background()

	var results []emailchecker.EmailCheckResult
	if c.String("file") != "" {
		results, err = processEmailsConcurrently(ctx, checker, emails)
		if err != nil {
			return err
		}
	} else {
		results, err = processEmailsSequentially(ctx, checker, emails)
		if err != nil {
			return err
		}
	}

	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %v", err)
	}

	fmt.Println(string(output))
	return nil
}

func processEmailsSequentially(ctx context.Context, checker *emailchecker.EmailChecker, emails []string) ([]emailchecker.EmailCheckResult, error) {
	var results []emailchecker.EmailCheckResult

	for _, email := range emails {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}

		params := emailchecker.EmailCheckParams{
			Email: email,
		}

		result, err := checker.Check(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to check email %s: %v", email, err)
		}

		results = append(results, result)
	}

	return results, nil
}

func processEmailsConcurrently(ctx context.Context, checker *emailchecker.EmailChecker, emails []string) ([]emailchecker.EmailCheckResult, error) {
	const maxGoroutines = 100

	var validEmails []string
	for _, email := range emails {
		email = strings.TrimSpace(email)
		if email != "" {
			validEmails = append(validEmails, email)
		}
	}

	if len(validEmails) == 0 {
		return []emailchecker.EmailCheckResult{}, nil
	}

	semaphore := make(chan struct{}, maxGoroutines)

	type indexedResult struct {
		index  int
		result emailchecker.EmailCheckResult
		err    error
	}

	resultsChan := make(chan indexedResult, len(validEmails))
	var wg sync.WaitGroup

	for i, email := range validEmails {
		wg.Add(1)
		go func(index int, emailAddr string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			params := emailchecker.EmailCheckParams{
				Email: emailAddr,
			}

			result, err := checker.Check(ctx, params)

			resultsChan <- indexedResult{
				index:  index,
				result: result,
				err:    err,
			}
		}(i, email)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	results := make([]emailchecker.EmailCheckResult, len(validEmails))
	var firstError error

	for indexedRes := range resultsChan {
		if indexedRes.err != nil && firstError == nil {
			firstError = fmt.Errorf("failed to check email %s: %v", validEmails[indexedRes.index], indexedRes.err)
		}

		results[indexedRes.index] = indexedRes.result
	}

	if firstError != nil {
		return nil, firstError
	}

	return results, nil
}

func startServer(c *cli.Context) error {
	checker, err := createChecker()
	if err != nil {
		return fmt.Errorf("failed to create checker: %v", err)
	}
	defer checker.Close()

	server := api.NewServer(checker, c.String("port"))
	return server.Start()
}

func readEmailsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return readEmailsFromReader(file)
}

func readEmailsFromStdin() ([]string, error) {
	return readEmailsFromReader(os.Stdin)
}

func readEmailsFromReader(reader io.Reader) ([]string, error) {
	var emails []string
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		email := strings.TrimSpace(scanner.Text())
		if email != "" {
			emails = append(emails, email)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return emails, nil
}
