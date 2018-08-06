package main

import (
	"crypto/tls"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ldap.v2"
	"strconv"
)

type LDAPConnection struct {
	app     *PlugApplication
	con     *ldap.Conn
	host    string
	bind_dn string
	bind_pw string
}

func (c *LDAPConnection) Init(
	app *PlugApplication,
	host,
	bind_dn,
	bind_pw string) {

	c.app = app
	c.host = host
	c.bind_dn = bind_dn
	c.bind_pw = bind_pw

	c.reconnectToLDAP()
}

func (c *LDAPConnection) reconnectToLDAP() {
	lcon, err := ldap.DialTLS("tcp", c.host,
		&tls.Config{ServerName: "ldap.csh.rit.edu"})
	if err != nil {
		c.app.db.AddLog(0, "ldap connection error: "+err.Error())
		log.Fatal(err)
	}
	err = lcon.Bind(c.bind_dn, c.bind_pw)
	if err != nil {
		c.app.db.AddLog(0, "ldap bind error: "+err.Error())
		log.Fatal(err)
	}
	c.con = lcon
}

func (c LDAPConnection) pingLDAPAlive() {
	searchReq := ldap.NewSearchRequest(
		"dc=csh,dc=rit,dc=edu",
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=top)",
		[]string{"dn"},
		nil,
	)
	_, err := c.con.Search(searchReq)
	if err != nil {
		c.reconnectToLDAP()
	}
}

func (c LDAPConnection) CheckIfAdmin(username string) bool {
	c.pingLDAPAlive()
	searchRequest := ldap.NewSearchRequest(
		"cn=users,cn=accounts,dc=csh,dc=rit,dc=edu",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(&(|(memberof=cn=drink,cn=groups,cn=accounts,dc=csh,dc=rit,dc=edu)(memberof=cn=rtp,cn=groups,cn=accounts,dc=csh,dc=rit,dc=edu)(memberof=cn=eboard,cn=groups,cn=accounts,dc=csh,dc=rit,dc=edu))(uid="+username+"))",
		[]string{"uid"},
		nil,
	)

	sr, err := c.con.Search(searchRequest)
	if err != nil {
		c.app.db.AddLog(0, "ldap search error: "+err.Error())
		log.Fatal(err)
		return false
	}
	return len(sr.Entries) > 0
}

func (c LDAPConnection) DecrementCredits(username string, credits int) bool {
	c.pingLDAPAlive()
	searchRequest := ldap.NewSearchRequest(
		"uid="+username+",cn=users,cn=accounts,dc=csh,dc=rit,dc=edu",
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)",
		[]string{"drinkBalance"},
		nil,
	)

	sr, err := c.con.Search(searchRequest)
	if err != nil {
		c.app.db.AddLog(0, "ldap search error: "+err.Error())
		log.Fatal(err)
	}

	balance, err := strconv.Atoi(sr.Entries[0].GetAttributeValue("drinkBalance"))
	if err != nil {
		c.app.db.AddLog(0, "ldap result parse error: "+err.Error())
		log.Fatal(err)
	}
	log.Info("current balance for %s is %d", username, balance)

	newBalance := balance - credits

	if newBalance < 0 {
		log.Info("Insufficient Credits! %d", balance)
		return false
	}

	modifyRequest := ldap.NewModifyRequest("uid=" + username + ",cn=users,cn=accounts,dc=csh,dc=rit,dc=edu")
	modifyRequest.Replace("drinkBalance", []string{fmt.Sprintf("%d", newBalance)})
	err = c.con.Modify(modifyRequest)
	if err != nil {
		c.app.db.AddLog(0, "ldap modification error: "+err.Error())
		log.Fatal(err)
	}
	log.Info("current balance for %s is %d", username, newBalance)

	return true
}
