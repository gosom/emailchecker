package dns

import (
	"net"
	"strings"

	"github.com/yl2chen/cidranger"
	"golang.org/x/net/publicsuffix"
)

type parkedDomainChecker struct {
	ranger cidranger.Ranger
}

func newParkedDomainChecker() *parkedDomainChecker {
	checker := parkedDomainChecker{
		ranger: cidranger.NewPCTrieRanger(),
	}

	for _, cidr := range parkedDomainIPsList {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}

		err = checker.ranger.Insert(cidranger.NewBasicRangerEntry(*network))
		if err != nil {
			panic(err)
		}
	}

	return &checker
}

func (p *parkedDomainChecker) IsParkedDomainIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	contains, err := p.ranger.Contains(ip)
	if err != nil {
		return false
	}

	return contains
}

func (p *parkedDomainChecker) IsParkedDomainNS(domain string) bool {
	domain = strings.TrimSuffix(domain, ".")
	baseDomain := extractBaseDomain(domain)

	return parkedDomansNS[baseDomain] || parkedDomansNS[domain]
}

func extractBaseDomain(domain string) string {
	baseDomain, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return domain
	}

	return baseDomain
}

// TODO: Move this to the database
var parkedDomansNS = map[string]bool{
	"above.com":                   true,
	"afternic.com":                true,
	"alter.com":                   true,
	"bodis.com":                   true,
	"bookmyname.com":              true,
	"brainydns.com":               true,
	"brandbucket.com":             true,
	"chookdns.com":                true,
	"cnomy.com":                   true,
	"commonmx.com":                true,
	"dan.com":                     true,
	"day.biz":                     true,
	"dingodns.com":                true,
	"directnic.com":               true,
	"dne.com":                     true,
	"dnslink.com":                 true,
	"dnsnuts.com":                 true,
	"dnsowl.com":                  true,
	"dnsspark.com":                true,
	"domain-for-sale.at":          true,
	"domain-for-sale.se":          true,
	"domaincntrol.com":            true,
	"domainhasexpired.com":        true,
	"domainist.com":               true,
	"domainmarket.com":            true,
	"domainmx.com":                true,
	"domainorderdns.nl":           true,
	"domainparking.ru":            true,
	"domainprofi.de":              true,
	"domainrecover.com":           true,
	"dsredirection.com":           true,
	"dsredirects.com":             true,
	"eftydns.com":                 true,
	"emailverification.info":      true,
	"emu-dns.com":                 true,
	"expiereddnsmanager.com":      true,
	"expirationwarning.net":       true,
	"expired.uniregistry-dns.com": true,
	"fabulous.com":                true,
	"failed-whois-verification.namecheap.com": true,
	"fastpark.net":            true,
	"freenom.com":             true,
	"gname.net":               true,
	"hastydns.com":            true,
	"hostresolver.com":        true,
	"ibspark.com":             true,
	"kirklanddc.com":          true,
	"koaladns.com":            true,
	"magpiedns.com":           true,
	"malkm.com":               true,
	"markmonitor.com":         true,
	"mijndomein.nl":           true,
	"milesmx.com":             true,
	"mytrafficmanagement.com": true,
	"name.com":                true,
	"namedynamics.net":        true,
	"nameprovider.net":        true,
	"ndsplitter.com":          true,
	"ns01.cashparking.com":    true,
	"ns02.cashparking.com":    true,
	"ns1.domain-is-4-sale-at-domainmarket.com": true,
	"ns1.domain.io":                        true,
	"ns1.namefind.com":                     true,
	"ns1.park.do":                          true,
	"ns1.pql.net":                          true,
	"ns1.smartname.com":                    true,
	"ns1.sonexo.eu":                        true,
	"ns1.undeveloped.com":                  true,
	"ns2.domain.io":                        true,
	"ns2.domainmarket.com":                 true,
	"ns2.namefind.com":                     true,
	"ns2.park.do":                          true,
	"ns2.pql.net":                          true,
	"ns2.smartname.com":                    true,
	"ns2.sonexo.com":                       true,
	"ns2.undeveloped.com":                  true,
	"ns3.tppns.com":                        true,
	"ns4.tppns.com":                        true,
	"nsresolution.com":                     true,
	"one.com":                              true,
	"onlydomains.com":                      true,
	"panamans.com":                         true,
	"park1.encirca.net":                    true,
	"park2.encirca.net":                    true,
	"parkdns1.internetvikings.com":         true,
	"parkdns2.internetvikings.com":         true,
	"parking-page.net":                     true,
	"parking.namecheap.com":                true,
	"parking1.ovh.net":                     true,
	"parking2.ovh.net":                     true,
	"parkingcrew.net":                      true,
	"parkingpage.namecheap.com":            true,
	"parkingspa.com":                       true,
	"parklogic.com":                        true,
	"parktons.com":                         true,
	"perfectdomain.com":                    true,
	"quokkadns.com":                        true,
	"redirectdom.com":                      true,
	"redmonddc.com":                        true,
	"registrar-servers.com":                true,
	"renewyourname.net":                    true,
	"rentondc.com":                         true,
	"rookdns.com":                          true,
	"rzone.de":                             true,
	"sav.com":                              true,
	"searchfusion.com":                     true,
	"searchreinvented.com":                 true,
	"securetrafficrouting.com":             true,
	"sedo.com":                             true,
	"sedoparking.com":                      true,
	"smtmdns.com":                          true,
	"snparking.ru":                         true,
	"squadhelp.com":                        true,
	"sslparking.com":                       true,
	"tacomadc.com":                         true,
	"taipandns.com":                        true,
	"thednscloud.com":                      true,
	"torresdns.com":                        true,
	"trafficcontrolrouter.com":             true,
	"trustednam.es":                        true,
	"uniregistrymarket.link":               true,
	"verify-contact-details.namecheap.com": true,
	"voodoo.com":                           true,
	"weaponizedcow.com":                    true,
	"wombatdns.com":                        true,
	"wordpress.com":                        true,
	"www.undeveloped.com----type.in":       true,
	"your-browser.this-domain.eu":          true,
	"ztomy.com":                            true,
}

var parkedDomainIPsList = []string{
	"103.120.80.111/32",
	"103.139.0.32/32",
	"103.224.182.0/23",
	"103.224.212.0/23",
	"104.26.6.37/32",
	"104.26.7.37/32",
	"119.28.128.52/32",
	"121.254.178.252/32",
	"13.225.34.0/24",
	"13.227.219.0/24",
	"13.248.216.40/32",
	"135.148.9.101/32",
	"141.8.224.195/32",
	"158.247.7.206/32",
	"158.69.201.47/32",
	"159.89.244.183/32",
	"164.90.244.158/32",
	"172.67.70.191/32",
	"18.164.52.0/24",
	"185.134.245.113/32",
	"185.53.176.0/22",
	"188.93.95.11/32",
	"192.185.0.218/32",
	"192.64.147.0/24",
	"194.58.112.165/32",
	"194.58.112.174/32",
	"198.54.117.192/26",
	"199.191.50.0/24",
	"199.58.179.10/32",
	"199.59.240.0/22",
	"2.57.90.16/32",
	"204.11.56.0/23",
	"207.148.248.143/32",
	"207.148.248.145/32",
	"208.91.196.0/23",
	"208.91.196.46/32",
	"208.91.197.46/32",
	"208.91.197.91/32",
	"209.99.40.222/32",
	"209.99.64.0/24",
	"213.145.228.16/32",
	"213.171.195.105/32",
	"216.40.34.41/32",
	"217.160.141.142/32",
	"217.160.95.94/32",
	"217.26.48.101/32",
	"217.70.184.38/32",
	"217.70.184.50/32",
	"3.139.159.151/32",
	"3.234.55.179/32",
	"3.64.163.50/32",
	"31.186.11.254/32",
	"31.31.205.163/32",
	"34.102.136.180/32",
	"34.102.221.37/32",
	"34.98.99.30/32",
	"35.186.238.101/32",
	"35.227.197.36/32",
	"37.97.254.27/32",
	"43.128.56.249/32",
	"45.79.222.138/32",
	"45.88.202.115/32",
	"46.28.105.2/32",
	"46.30.211.38/32",
	"46.4.13.97/32",
	"46.8.8.100/32",
	"47.91.170.222/32",
	"5.9.161.60/32",
	"50.28.32.8/32",
	"52.128.23.153/32",
	"52.222.139.0/24",
	"52.222.149.0/24",
	"52.222.158.0/24",
	"52.222.174.0/24",
	"52.58.78.16/32",
	"52.60.87.163/32",
	"52.84.174.0/24",
	"62.149.128.40/32",
	"64.190.62.0/23",
	"64.70.19.203/32",
	"64.70.19.98/32",
	"66.81.199.0/24",
	"74.220.199.14/32",
	"74.220.199.15/32",
	"74.220.199.6/32",
	"74.220.199.8/32",
	"74.220.199.9/32",
	"75.2.115.196/32",
	"75.2.18.233/32",
	"75.2.26.18/32",
	"76.223.65.111/32",
	"78.47.145.38/32",
	"81.2.194.128/32",
	"88.198.29.97/32",
	"91.184.0.100/32",
	"91.195.240.0/23",
	"91.195.240.80/28",
	"93.191.168.52/32",
	"94.136.40.51/32",
	"95.217.58.108/32",
	"98.124.204.16/32",
	"99.83.154.118/32",
}
