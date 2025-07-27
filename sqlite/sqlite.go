package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"emailchecker"

	"golang.org/x/net/publicsuffix"
	_ "modernc.org/sqlite"
)

type Repository struct {
	readDB  *sql.DB
	writeDB *sql.DB
}

func New(dbPath string) (*Repository, error) {
	connStr := fmt.Sprintf("%s?"+
		"_pragma=journal_mode(WAL)&"+
		"_pragma=synchronous(NORMAL)&"+
		"_pragma=cache_size(-64000)&"+
		"_pragma=temp_store(MEMORY)&"+
		"_pragma=mmap_size(268435456)&"+
		"_pragma=page_size(4096)&"+
		"_pragma=wal_autocheckpoint(1000)&"+
		"_pragma=busy_timeout(30000)&"+
		"_pragma=foreign_keys(ON)&"+
		"_pragma=optimize",
		dbPath)

	readDB, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := testConnection(context.Background(), readDB); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	readDB.SetMaxOpenConns(8)
	readDB.SetMaxIdleConns(2)
	readDB.SetConnMaxLifetime(1 * time.Hour)
	readDB.SetConnMaxIdleTime(10 * time.Minute)

	writeDB, err := sql.Open("sqlite", connStr)
	if err != nil {
		readDB.Close()
		return nil, fmt.Errorf("failed to open database for writing: %w", err)
	}

	if err := testConnection(context.Background(), writeDB); err != nil {
		readDB.Close()

		return nil, fmt.Errorf("failed to connect to database for writing: %w", err)
	}

	writeDB.SetMaxOpenConns(1)
	writeDB.SetMaxIdleConns(1)
	writeDB.SetConnMaxLifetime(0)
	writeDB.SetConnMaxIdleTime(30 * time.Minute)

	repo := &Repository{
		readDB:  readDB,
		writeDB: writeDB,
	}

	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelMigrate()

	if err := repo.init(migrateCtx); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *Repository) Close() {
	r.readDB.Close()
}

func (r *Repository) IsDisposable(ctx context.Context, domain string) (bool, error) {
	var exists bool
	domain = strings.TrimSuffix(domain, ".")

	baseDomain := extractBaseDomain(domain)

	if domain == baseDomain {
		query := "SELECT EXISTS(SELECT 1 FROM disposable_domains WHERE domain = ?)"

		err := r.readDB.QueryRowContext(ctx, query, domain).Scan(&exists)

		if err != nil && err != sql.ErrNoRows {
			return false, fmt.Errorf("could not query domain: %w", err)
		}

		return exists, nil
	}

	query := "SELECT EXISTS(SELECT 1 FROM disposable_domains WHERE domain IN (?, ?))"
	err := r.readDB.QueryRowContext(ctx, query, domain, baseDomain).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("could not query domain: %w", err)
	}

	return exists, nil
}

func (r *Repository) UpdateDomains(ctx context.Context, newDomains []string) error {
	mainTable := "disposable_domains"
	key := "last_refresh_at"

	return r.updateDomains(ctx, updateDomainsParams{
		Domains:   newDomains,
		MainTable: mainTable,
		Key:       key,
	})
}

func (r *Repository) NeedsRefresh(ctx context.Context) (bool, error) {
	return r.needsRefresh(ctx, "last_refresh_at")
}

func (r *Repository) GetDNSRecord(ctx context.Context, domain string) (*emailchecker.DNSRecord, error) {
	var record emailchecker.DNSRecord
	record.Domain = domain

	query := "SELECT data, created_at FROM dns_records WHERE domain = ?"
	err := r.readDB.QueryRowContext(ctx, query, domain).Scan(&record.Data, &record.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get DNS record for '%s': %w", domain, err)
	}

	return &record, nil
}

func (r *Repository) UpsertDNSRecord(ctx context.Context, domain string, data []byte) error {
	query := `
	INSERT INTO dns_records (domain, data, created_at)
	VALUES (?, ?, ?)
	ON CONFLICT(domain) DO UPDATE SET
		data = excluded.data,
		created_at = excluded.created_at;
	`
	_, err := r.writeDB.ExecContext(ctx, query, domain, data, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("could not upsert DNS record for '%s': %w", domain, err)
	}
	return nil
}

func (r *Repository) IsTop(ctx context.Context, domain string) (bool, error) {
	var exists bool
	domain = strings.TrimSuffix(domain, ".")

	query := "SELECT EXISTS(SELECT 1 FROM top_domains WHERE domain = ?)"

	err := r.readDB.QueryRowContext(ctx, query, domain).Scan(&exists)

	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("could not query domain: %w", err)
	}

	return exists, nil
}

func (r *Repository) TopNeedsRefresh(ctx context.Context) (bool, error) {
	return r.needsRefresh(ctx, "top_domains_refreshed_at")
}

func (r *Repository) UpdateTopDomains(ctx context.Context, domains []string) error {
	return r.updateDomains(ctx, updateDomainsParams{
		Domains:   domains,
		MainTable: "top_domains",
		Key:       "top_domains_refreshed_at",
	})
}

func (r *Repository) IsEducationalDomain(ctx context.Context, domain string) (bool, error) {
	var exists bool
	domain = strings.TrimSuffix(domain, ".")

	query := "SELECT EXISTS(SELECT 1 FROM edu_domains WHERE domain = ?)"

	err := r.readDB.QueryRowContext(ctx, query, domain).Scan(&exists)

	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("could not query domain: %w", err)
	}

	return exists, nil
}

func (r *Repository) UpdateEducationalDomains(ctx context.Context, domains []string) error {
	return r.updateDomains(ctx, updateDomainsParams{
		Domains:   domains,
		MainTable: "edu_domains",
		Key:       "edu_domains_refreshed_at",
	})
}

func (r *Repository) NeedsEduRefresh(ctx context.Context) (bool, error) {
	return r.needsRefresh(ctx, "edu_domains_refreshed_at")
}

type updateDomainsParams struct {
	Domains   []string
	MainTable string
	Key       string
}

func (r *Repository) updateDomains(ctx context.Context, params updateDomainsParams) error {
	mainTable := params.MainTable
	newTable := fmt.Sprintf("%s_new", mainTable)
	oldTable := fmt.Sprintf("%s_old", mainTable)

	tx, err := r.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	err = r.createDisposableDomainsTable(ctx, tx, newTable)
	if err != nil {
		return fmt.Errorf("could not create new table '%s': %w", newTable, err)
	}

	stmt, err := tx.Prepare(fmt.Sprintf("INSERT INTO %s (domain) VALUES (?)", newTable))
	if err != nil {
		return fmt.Errorf("could not prepare insert for new table: %w", err)
	}
	defer stmt.Close()

	seen := make(map[string]struct{}, len(params.Domains))
	for _, domain := range params.Domains {
		if domain == "" {
			continue
		}

		if _, ok := seen[domain]; ok {
			continue
		}

		seen[domain] = struct{}{}
		if _, err := stmt.Exec(domain); err != nil {
			return fmt.Errorf("could not insert domain '%s' into new table: %w", domain, err)
		}
	}
	renameOldCmd := fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", mainTable, oldTable)
	if _, err := tx.Exec(renameOldCmd); err != nil {
		return fmt.Errorf("could not rename main table to old: %w", err)
	}

	renameNewCmd := fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", newTable, mainTable)
	if _, err := tx.Exec(renameNewCmd); err != nil {
		return fmt.Errorf("could not rename new table to main: %w", err)
	}
	dropOldCmd := fmt.Sprintf("DROP TABLE %s;", oldTable)
	if _, err := tx.Exec(dropOldCmd); err != nil {
		return fmt.Errorf("could not drop old table: %w", err)
	}

	err = r.updateRefreshTimestamp(ctx, tx, params.Key)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) needsRefresh(ctx context.Context, key string) (bool, error) {
	var lastRefreshStr string

	query := "SELECT value FROM app_metadata WHERE key = ?"
	err := r.readDB.QueryRowContext(ctx, query, key).Scan(&lastRefreshStr)

	switch {
	case err == sql.ErrNoRows:
		return true, nil

	case err != nil:
		return false, fmt.Errorf("could not query last refresh time: %w", err)
	}

	lastRefreshTime, err := time.Parse(time.RFC3339, lastRefreshStr)
	if err != nil {
		return true, fmt.Errorf("could not parse last refresh time '%s': %w", lastRefreshStr, err)
	}

	durationSinceRefresh := time.Since(lastRefreshTime)
	if durationSinceRefresh > 12*time.Hour {
		return true, nil
	}

	return false, nil
}

func (r *Repository) updateRefreshTimestamp(ctx context.Context, tx *sql.Tx, key string) error {
	query := `
	INSERT INTO app_metadata (key, value)
	VALUES (?, ?)
	ON CONFLICT(key) DO UPDATE SET value = excluded.value;
	`
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := tx.ExecContext(ctx, query, key, now)
	if err != nil {
		return fmt.Errorf("could not update refresh timestamp: %w", err)
	}

	return nil
}

func (r *Repository) init(ctx context.Context) error {
	tx, err := r.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	err = r.createMetadataTable(ctx, tx)
	if err != nil {
		return fmt.Errorf("could not create disposable_domains_refreshed_at table: %w", err)
	}

	err = r.createDisposableDomainsTable(ctx, tx, "disposable_domains")
	if err != nil {
		return fmt.Errorf("could not create disposable_domains table: %w", err)
	}

	err = r.createDNSRecordsTable(ctx, tx)
	if err != nil {
		return fmt.Errorf("could not create dns_records table: %w", err)
	}

	err = r.createTopDomainsTable(ctx, tx)
	if err != nil {
		return fmt.Errorf("could not create top_domains table: %w", err)
	}

	err = r.createEduDomainsTable(ctx, tx)
	if err != nil {
		return fmt.Errorf("could not create edu_domains table: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) createDisposableDomainsTable(ctx context.Context, tx *sql.Tx, name string) error {
	schema := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			domain TEXT PRIMARY KEY NOT NULL
	);`, name)

	_, err := tx.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("could not create disposable_domains table: %w", err)
	}

	return nil
}

func (r *Repository) createMetadataTable(ctx context.Context, tx *sql.Tx) error {
	schema := `
	CREATE TABLE IF NOT EXISTS app_metadata (
		key TEXT PRIMARY KEY NOT NULL,
		value TEXT NOT NULL
	);`
	_, err := tx.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("could not create app_metadata table: %w", err)
	}
	return nil
}

func (r *Repository) createDNSRecordsTable(ctx context.Context, tx *sql.Tx) error {
	schema := `
	CREATE TABLE IF NOT EXISTS dns_records (
		domain TEXT PRIMARY KEY NOT NULL,
		data BLOB NOT NULL,
		created_at TIMESTAMP NOT NULL
	);`
	_, err := tx.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("could not create dns_records table: %w", err)
	}
	return nil
}

func (r *Repository) createTopDomainsTable(ctx context.Context, tx *sql.Tx) error {
	schema := `
	CREATE TABLE IF NOT EXISTS top_domains (
		domain TEXT PRIMARY KEY NOT NULL
	);`
	_, err := tx.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("could not create top_domains table: %w", err)
	}
	return nil
}

func (r *Repository) createEduDomainsTable(ctx context.Context, tx *sql.Tx) error {
	schema := `
	CREATE TABLE IF NOT EXISTS edu_domains (
		domain TEXT PRIMARY KEY NOT NULL
	);`
	_, err := tx.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("could not create edu_domains table: %w", err)
	}
	return nil
}

func extractBaseDomain(domain string) string {
	baseDomain, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return domain
	}

	return baseDomain
}

func testConnection(ctx context.Context, conn *sql.DB) error {
	pingCtx, cancelPing := context.WithTimeout(ctx, 2*time.Second)
	defer cancelPing()

	if err := conn.PingContext(pingCtx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}
