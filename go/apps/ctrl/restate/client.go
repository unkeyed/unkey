package restate

import (
	"context"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
)

type Workflow[I any] interface {
	Run(ctx restate.WorkflowContext, input I) error
}

type Runner[I any] struct {
	name   string
	client *restateingress.Client
}

func CreateRunner[I any](c *restateingress.Client, workflow Workflow[I]) *Runner[I] {
	rf := restate.Reflect(workflow)
	return &Runner[I]{
		name:   rf.Name(),
		client: c,
	}

}

func (r *Runner[I]) Run(ctx context.Context, id string, input I) restateingress.Invocation {
	return restateingress.Workflow[I, any](r.client, r.name, id, "Run").Send(ctx, input)
}
