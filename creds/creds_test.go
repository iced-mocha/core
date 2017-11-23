package creds

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/stretchr/testify/suite"
)

type CredsTestSuite struct {
	suite.Suite
}

func (s *CredsTestSuite) SetupSuite() {
	log.SetOutput(ioutil.Discard)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(CredsTestSuite))
}
