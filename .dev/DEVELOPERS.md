# DEVELOPERS.md

## LDAP
Run `ldapadd -x -H ldap://localhost:389 -D "cn=admin,dc=example,dc=org" -w admin -f sample.ldif` to import the test environment into the LDAP container.