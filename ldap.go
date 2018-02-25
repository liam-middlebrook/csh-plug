package main

import (
	"crypto/tls"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ldap.v2"
	"os"
	"strconv"
)

var con *ldap.Conn
var ldap_bind_env_var_name string
var ldap_bindpw_env_var_name string
var ldap_host_env_var_name string

func reconnectToLDAP() *ldap.Conn {
	lcon, err := ldap.DialTLS("tcp", os.Getenv(ldap_host_env_var_name),
		&tls.Config{ServerName: "ldap.csh.rit.edu"})
	if err != nil {
		AddLog(0, "ldap connection error: "+err.Error())
		log.Fatal(err)
	}
	err = lcon.Bind(os.Getenv(ldap_bind_env_var_name), os.Getenv(ldap_bindpw_env_var_name))
	if err != nil {
		AddLog(0, "ldap bind error: "+err.Error())
		log.Fatal(err)
	}
	return lcon
}

func pingLDAPAlive() {
	searchReq := ldap.NewSearchRequest(
		"dc=csh,dc=rit,dc=edu",
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=top)",
		[]string{"dn"},
		nil,
	)
	_, err := con.Search(searchReq)
	if err != nil {
		con = reconnectToLDAP()
	}
}

func LDAPInit(host, binddn, bindpw string) {
	ldap_bind_env_var_name = binddn
	ldap_bindpw_env_var_name = bindpw
	ldap_host_env_var_name = host
	con = reconnectToLDAP()
}

func CheckIfAdmin(username string) bool {
	pingLDAPAlive()
	searchRequest := ldap.NewSearchRequest(
		"cn=users,cn=accounts,dc=csh,dc=rit,dc=edu",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(&(|(memberof=cn=drink,cn=groups,cn=accounts,dc=csh,dc=rit,dc=edu)(memberof=cn=rtp,cn=groups,cn=accounts,dc=csh,dc=rit,dc=edu)(memberof=cn=eboard,cn=groups,cn=accounts,dc=csh,dc=rit,dc=edu))(uid="+username+"))",
		[]string{"uid"},
		nil,
	)

	sr, err := con.Search(searchRequest)
	if err != nil {
		AddLog(0, "ldap search error: "+err.Error())
		log.Fatal(err)
		return false
	}
	return len(sr.Entries) > 0
}

func DecrementCredits(username string, credits int) bool {
	pingLDAPAlive()
	searchRequest := ldap.NewSearchRequest(
		"uid="+username+",cn=users,cn=accounts,dc=csh,dc=rit,dc=edu",
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)",
		[]string{"drinkBalance"},
		nil,
	)

	sr, err := con.Search(searchRequest)
	if err != nil {
		AddLog(0, "ldap search error: "+err.Error())
		log.Fatal(err)
	}

	balance, err := strconv.Atoi(sr.Entries[0].GetAttributeValue("drinkBalance"))
	if err != nil {
		AddLog(0, "ldap result parse error: "+err.Error())
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
	err = con.Modify(modifyRequest)
	if err != nil {
		AddLog(0, "ldap modification error: "+err.Error())
		log.Fatal(err)
	}
	log.Info("current balance for %s is %d", username, newBalance)

	return true
}
