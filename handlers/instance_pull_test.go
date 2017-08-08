package handlers

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rancher/agent/utils"
	"golang.org/x/net/context"
	check "gopkg.in/check.v1"
)

func (s *EventTestSuite) TestPullImage(c *check.C) {
	cli := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	cli.ImageRemove(context.Background(), "quay.io/strongmonkey/echo-hello:latestrandom", types.ImageRemoveOptions{
		PruneChildren: true,
	})

	rawEvent := loadEvent("./test_events/instance_pull", c)
	reply := testEvent(rawEvent, c)

	imageInspect, _, err := cli.ImageInspectWithRaw(context.Background(), "quay.io/strongmonkey/echo-hello:latestrandom")
	if err != nil {
		c.Fatal(err)
	}

	c.Assert(reply.Transitioning != "error", check.Equals, true)
	value, ok := utils.GetFieldsIfExist(reply.Data, "fields", "dockerImage")
	if !ok {
		c.Fatal("can't find image inspect")
	}
	c.Assert(value.(types.ImageInspect).ID, check.Equals, imageInspect.ID)

	event := unmarshalEvent(rawEvent, c)
	event["data"] = map[string]interface{}{
		"instancePull": map[string]interface{}{
			"kind": "docker",
			"image": map[string]interface{}{
				"data": map[string]interface{}{
					"dockerImage": map[string]interface{}{
						"fullName": "strongmonkey/echo-hello:latest",
						"server":   "quay.io",
					},
				},
			},
			"mode":     "all",
			"complete": true,
			"tag":      "random",
		},
	}

	rawEvent = marshalEvent(event, c)
	reply = testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)
	_, _, err = cli.ImageInspectWithRaw(context.Background(), "quay.io/strongmonkey/echo-hello:latestrandom")
	c.Assert(client.IsErrImageNotFound(err), check.Equals, true)

	event = unmarshalEvent(rawEvent, c)
	event["data"] = map[string]interface{}{
		"instancePull": map[string]interface{}{
			"kind": "docker",
			"image": map[string]interface{}{
				"data": map[string]interface{}{
					"dockerImage": map[string]interface{}{
						"fullName": "invalidimage%!@#$%^&",
						"server":   "quay.io",
					},
				},
			},
			"mode":     "cached",
			"complete": true,
			"tag":      "random",
		},
	}

	rawEvent = marshalEvent(event, c)
	reply = testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)
}
