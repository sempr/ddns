package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"github.com/sempr/ddns"
)

type APIServer struct {
	d      ddns.DNSStorage
	prefix string
	domain string
	srv    *http.Server
}

func NewAPIServer(d ddns.DNSStorage, prefix string, domain string) *APIServer {
	return &APIServer{d: d, prefix: prefix, domain: domain}
}

func (s *APIServer) home(c *gin.Context) {
	c.HTML(200, "index.html", gin.H{"domain": s.domain})
}

func (s *APIServer) update(c *gin.Context) {
	hostname, valid := isValidHostname(c.Params.ByName("hostname"))
	token := c.Params.ByName("token")
	ctx := c.Request.Context()
	if !valid {
		c.JSON(404, gin.H{"error": "This hostname is not valid"})
		return
	}

	ipStrings := strings.Split(c.Query("myip"), "|")
	var ips []net.IP
	for _, ip_ := range ipStrings {
		if ip := net.ParseIP(ip_); ip != nil {
			ips = append(ips, ip)
		}
	}
	if len(ips) == 0 {
		ip, _ := extractRemoteAddr(c.Request)
		ips = append(ips, net.ParseIP(ip))
	}

	qType := dns.TypeA
	var records []string
	if strings.Contains(ips[0].String(), ":") {
		// v6
		qType = dns.TypeAAAA
		for _, ip_ := range ips {
			if strings.Contains(ip_.String(), ":") {
				records = append(records, ip_.String())
			}
		}

	} else {
		// v4
		for _, ip_ := range ips {
			if strings.Contains(ip_.String(), ".") {
				records = append(records, ip_.String())
			}
		}
	}

	records = s.d.Update(ctx, hostname, token, qType, records)
	if len(records) == 0 {
		c.JSON(404, gin.H{
			"error": "This hostname has not been registered or is expired.",
		})
		return
	}

	c.JSON(200, gin.H{
		"current_ip": records,
		"status":     "Successfuly updated",
	})
}

func (s *APIServer) new(c *gin.Context) {
	hostname, valid := isValidHostname(c.Params.ByName("hostname"))

	if !valid {
		c.JSON(404, gin.H{"error": "This hostname is not valid"})
		return
	}

	var ctx = c.Request.Context()

	if valid = s.d.Valid(ctx, hostname); !valid {
		c.JSON(403, gin.H{"error": "This hostname has already been registered."})
		return
	}

	token := s.d.New(ctx, hostname)

	if token == "" {
		c.JSON(400, gin.H{"error": "Could not register host."})
		return
	}

	c.JSON(200, gin.H{
		"hostname":    hostname,
		"token":       token,
		"update_link": fmt.Sprintf("/update/%s/%s", hostname, token),
	})
}

func (s *APIServer) available(c *gin.Context) {
	hostname, valid := isValidHostname(c.Params.ByName("hostname"))
	if valid {
		valid = s.d.Valid(c.Request.Context(), hostname)
	}
	c.JSON(200, gin.H{
		"available": valid,
	})
}

func (s *APIServer) delete(c *gin.Context) {

}

func (s *APIServer) Run(port int) error {
	r := gin.Default()
	r.SetHTMLTemplate(buildTemplate())
	r.GET("/", s.home)
	r.GET("/update/:hostname/:token", s.update)
	r.DELETE("/update/:hostname/:token", s.delete)
	r.GET("/new/:hostname", s.new)
	r.GET("/available/:hostname", s.available)
	log.Printf("start http server on port %d", port)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
	s.srv = srv

	if err := srv.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func (s *APIServer) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
