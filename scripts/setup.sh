#!/bin/bash

while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -f|--force)
    FORCE="YES"
    shift # past argument
    ;;
esac
done

# Verify's that the given dependency is installed, otherwises exits
installed_or_exit () {
	if  [[ !  -z  $(command -v ${1}) ]]
	 then
		# The package is installed
		echo -n ""
	 else
		echo "${1} is not installed, please install and rerun this script."
		echo "Installation instructions can be found here:"
		echo "		 ${2}"
		exit 1	
		# The package is not installed
	fi
}

# Verify the following are installed
installed_or_exit sqlite3 'https://www.sqlite.org/quickstart.html'

if [ -z ${GOPATH} ] 
 then 
	# GOPATH is not set so create database in current directory
	# In this case this script must be run inside of the /scripts directory
	database_file="database.db"
	init_database="initDB.sql"
 else 
	# GOPATH is set
	database_file="${GOPATH}/src/github.com/icedmocha/core/database.db"
	init_database="${GOPATH}/src/github.com/icedmocha/core/scripts/initDB.sql"
 fi

# Now lets check if our database file exists so we can prevent accidently overwriting it
if [ -e ${database_file} ]
then
    if [ -z ${FORCE} ]
	 then
		# If the force param is not set give them a warning and ask to rerun with -f (--force)
		echo "${database_file} already exists. Rerun with -f to delete existing database and start over. ALL DATA WILL BE LOST"
		exit 0
	fi
fi

# Remove existing database file if it already exists
rm ${database_file} 2> /dev/null

# Create our table in the database
sqlite3 ${database_file} < ${init_database}


