package test

import (
	//"fmt"
	//"github.com/Sirupsen/logrus"
	"gopkg.in/check.v1"
	"testing"
	//"github.com/rancher/agent/handlers/utils"
)

func init() {

}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type ComputeTestSuite struct {
}

var _ = check.Suite(&ComputeTestSuite{})

func (s *ComputeTestSuite) SetUpSuite(c *check.C) {
}

/*
func (s *ComputeTestSuite) TestInstanceActivate(c *check.C) {

	// Load the event to a byte array from the specified file
	rawEvent := loadEvent("../test_events/instance_activate", c)

	// Optional: you can unmarshal, modify, and marshal the event data if you need to. This is equivalent to the "pre"
	// functions in python-agent
	event := unmarshalEvent(rawEvent, c)
	instance := getInstance(event, c)
	event["replyTo"] = "new-reply-to"
	instance["name"] = "new-name"
	rawEvent = marshalEvent(event, c)


	// Run the event through the framework
	reply := testEvent(rawEvent, c)
	if reply != nil {
		logrus.Info(fmt.Sprintf("reply data: %v", reply.Data))
	}
	// Assert whatever you need to on the reply event. This is equivalent to the "post" functions in python-agent
	c.Assert(reply.Name, check.Equals, "new-reply-to")
	//c.Assert(reply.Data.(map[string]interface{}).["instance"].(map[string]interface{})[])
	// As an example, once you implement some more logic, you could verify that the reply has the instance name as "new-name"
	utils.DeleteContainer("/c861f990-4472-4fa1-960f-65171b544c28")
}
*/
