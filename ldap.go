package main

import (
	"crypto/tls"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ldap.v2"
	"strconv"
)

var con *ldap.Conn

func LDAPInit(host, binddn, bindpw string) {
	var err error
	con, err = ldap.DialTLS("tcp", host, &tls.Config{ServerName: "ldap.csh.rit.edu"})
	if err != nil {
		log.Fatal(err)
	}
	err = con.Bind(binddn, bindpw)
	if err != nil {
		log.Fatal(err)
	}
}

func DecrementCredits(username string, credits int) bool {
	searchRequest := ldap.NewSearchRequest(
		"uid="+username+",ou=Users,dc=csh,dc=rit,dc=edu",
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)",
		[]string{"drinkBalance"},
		nil,
	)

	sr, err := con.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	balance, err := strconv.Atoi(sr.Entries[0].GetAttributeValue("drinkBalance"))
	if err != nil {
		log.Fatal(err)
	}
	log.Info("current balance for %s is %d", username, balance)

	newBalance := balance - credits

	if newBalance < 0 {
		log.Info("Insufficient Credits! %d", balance)
		return false
	}

	modifyRequest := ldap.NewModifyRequest("uid=" + username + ",ou=Users,dc=csh,dc=rit,dc=edu")
	modifyRequest.Replace("drinkBalance", []string{fmt.Sprintf("%d", newBalance)})
	err = con.Modify(modifyRequest)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("current balance for %s is %d", username, newBalance)

	return true
}
