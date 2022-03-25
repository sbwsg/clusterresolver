/*
Copyright 2022 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	pipelinesclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	"github.com/tektoncd/resolution/pkg/common"
	"github.com/tektoncd/resolution/pkg/resolver/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/injection/sharedmain"
)

const clusterResolverPrivateNamespace = "tekton-cluster-scoped-resources"
const jsonContentType = "application/json"

func main() {
	sharedmain.Main("controller",
		framework.NewController(context.Background(), &resolver{}),
	)
}

type resolver struct {
	// The clientset used to look up tasks and pipelines from the
	// clusterresolver's private namespace.
	Pipelineclientset pipelinesclientset.Interface
}

// Initialize creates an instance of the pipelines clientset so that
// tasks and pipelines can be looked up.
func (r *resolver) Initialize(ctx context.Context) error {
	r.Pipelineclientset = pipelineclient.Get(ctx)
	return nil
}

// GetName returns a string name to refer to this resolver by.
func (r *resolver) GetName(context.Context) string {
	return "clusterresolver"
}

// GetSelector returns a map of labels to match requests to this resolver.
func (r *resolver) GetSelector(context.Context) map[string]string {
	return map[string]string{
		common.LabelKeyResolverType: "clusterresolver",
	}
}

// ValidateParams ensures parameters from a request are as expected.
// Only "kind" and "name" are needed.
func (r *resolver) ValidateParams(ctx context.Context, params map[string]string) error {
	if len(params) == 0 {
		return errors.New(`require "kind" and "name" params`)
	}
	kind, hasKind := params["kind"]
	if !hasKind {
		return errors.New(`require "kind" param`)
	}
	kind = strings.TrimSpace(strings.ToLower(kind))
	if kind != "task" && kind != "pipeline" {
		return fmt.Errorf("unrecognized kind %q, only task and pipeline are supported", kind)
	}
	if _, has := params["name"]; !has {
		return errors.New(`require "name" param`)
	}
	return nil
}

// Resolve uses the given params to resolve the requested file or resource.
func (r *resolver) Resolve(ctx context.Context, params map[string]string) (framework.ResolvedResource, error) {
	name := params["name"]
	switch params["kind"] {
	case "task":
		task, err := r.Pipelineclientset.TektonV1beta1().Tasks(clusterResolverPrivateNamespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting cluster-scoped task %q: %w", name, err)
		}
		// Strip unwanted fields like the k8s
		// last-applied-configuration.
		// simplifiedTask := pipelinesv1beta1.Task{
		// 	Metadata: metav1.ObjectMeta{
		// 		Name: task.Name,
		// 	},
		// 	Spec: task.Spec,
		// }
		return resolvedClusterScopedResource(task)
	case "pipeline":
		pipeline, err := r.Pipelineclientset.TektonV1beta1().Pipelines(clusterResolverPrivateNamespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting cluster-scoped pipeline %q: %w", name, err)
		}
		// Strip unwanted fields like the k8s
		// last-applied-configuration.
		// simplifiedPipeline := pipelinesv1beta1.Pipeline{
		// 	Metadata: metav1.ObjectMeta{
		// 		Name: pipeline.Name,
		// 	},
		// 	Spec: pipeline.Spec,
		// }
		return resolvedClusterScopedResource(pipeline)
	default:
	}
	return nil, fmt.Errorf("unrecognized cluster-scoped resource kind %q", params["kind"])
}

func resolvedClusterScopedResource(obj runtime.Object) (framework.ResolvedResource, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal resolved resource to json: %w", err)
	}
	return &resolvedResource{
		data: data,
	}, nil
}

// resolvedResource wraps the data we want to return to Pipelines
type resolvedResource struct {
	data []byte
}

// Data returns the bytes of the task or pipeline resolved from the
// private namespace.
func (r *resolvedResource) Data() []byte {
	return r.data
}

// Annotations returns a content-type of json since the data is
// serialized as json.
func (r *resolvedResource) Annotations() map[string]string {
	return map[string]string{
		common.AnnotationKeyContentType: jsonContentType,
	}
}
