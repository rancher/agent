package handlers

import check "gopkg.in/check.v1"

func (s *EventTestSuite) TestPullImage(c *check.C) {
	rawEvent := loadEvent("./test_events/instance_pull", c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)
}
