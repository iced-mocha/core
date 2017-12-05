package sql

import (
	"log"
	"testing"

	"github.com/stretchr/testify/suite"
)

const (
	testDatabaseName = "test.db"
)

type DriverTestSuite struct {
	suite.Suite
	d *driver
}

// Deletes all data from database for testing
func (suite *DriverTestSuite) WipeData() {
	// Query to delete all entries from UserInfo table
	_, err := suite.d.db.Exec("DELETE FROM UserInfo")
	suite.Nil(err)
}

func (suite *DriverTestSuite) SetupSuite() {
	var err error

	// Note it is assumed an initialized test.db file exists within the test/ directory
	suite.d, err = New(Config{DatabasePath: testDatabaseName})
	if err != nil {
		log.Printf("Unable to create db object: %v\n", err)
	}
}

func (suite *DriverTestSuite) SetupTest() {
	// Before every test wipe the database and reset it
	suite.WipeData()

}

func (suite *DriverTestSuite) TestUsernameExists() {
	// An empty database should not contain any usernames
	exists, err := suite.d.UsernameExists("")
	suite.Nil(err)
	suite.False(exists)

	exists, err = suite.d.UsernameExists("jgore")
	suite.Nil(err)
	suite.False(exists)
}

func (suite *DriverTestSuite) TestNew() {
	// Creating a basic driver should work so long as the file is there
	_, err := New(Config{})
	if err != nil {
		log.Fatalf("Unable to create driver: %v\n", err)
	}
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(DriverTestSuite))
}
