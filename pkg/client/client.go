/*
Copyright 2026.

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

// Package client provides typed clients for Gas Town CRDs.
package client

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

var (
	// GVR definitions for Gas Town CRDs
	RigGVR = schema.GroupVersionResource{
		Group:    "gastown.gastown.io",
		Version:  "v1alpha1",
		Resource: "rigs",
	}

	PolecatGVR = schema.GroupVersionResource{
		Group:    "gastown.gastown.io",
		Version:  "v1alpha1",
		Resource: "polecats",
	}

	ConvoyGVR = schema.GroupVersionResource{
		Group:    "gastown.gastown.io",
		Version:  "v1alpha1",
		Resource: "convoys",
	}
)

// Client provides typed access to Gas Town CRDs
type Client struct {
	dynamic dynamic.Interface
	scheme  *runtime.Scheme
}

// NewClient creates a new Gas Town client from a REST config
func NewClient(config *rest.Config) (*Client, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := gastownv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add gastown types to scheme: %w", err)
	}

	return &Client{
		dynamic: dynamicClient,
		scheme:  scheme,
	}, nil
}

// NewClientFromDynamic creates a Gas Town client from an existing dynamic client
func NewClientFromDynamic(dynamicClient dynamic.Interface) *Client {
	scheme := runtime.NewScheme()
	_ = gastownv1alpha1.AddToScheme(scheme)

	return &Client{
		dynamic: dynamicClient,
		scheme:  scheme,
	}
}

// Rigs returns a RigClient for cluster-scoped Rig resources
func (c *Client) Rigs() RigClient {
	return &rigClient{
		client: c.dynamic.Resource(RigGVR),
		scheme: c.scheme,
	}
}

// Polecats returns a PolecatClient for the given namespace
func (c *Client) Polecats(namespace string) PolecatClient {
	return &polecatClient{
		client:    c.dynamic.Resource(PolecatGVR).Namespace(namespace),
		scheme:    c.scheme,
		namespace: namespace,
	}
}

// Convoys returns a ConvoyClient for the given namespace
func (c *Client) Convoys(namespace string) ConvoyClient {
	return &convoyClient{
		client:    c.dynamic.Resource(ConvoyGVR).Namespace(namespace),
		scheme:    c.scheme,
		namespace: namespace,
	}
}

// RigClient provides typed operations for Rig resources
type RigClient interface {
	Get(ctx context.Context, name string) (*gastownv1alpha1.Rig, error)
	List(ctx context.Context, opts metav1.ListOptions) (*gastownv1alpha1.RigList, error)
	Create(ctx context.Context, rig *gastownv1alpha1.Rig) (*gastownv1alpha1.Rig, error)
	Update(ctx context.Context, rig *gastownv1alpha1.Rig) (*gastownv1alpha1.Rig, error)
	UpdateStatus(ctx context.Context, rig *gastownv1alpha1.Rig) (*gastownv1alpha1.Rig, error)
	Delete(ctx context.Context, name string) error
}

// PolecatClient provides typed operations for Polecat resources
type PolecatClient interface {
	Get(ctx context.Context, name string) (*gastownv1alpha1.Polecat, error)
	List(ctx context.Context, opts metav1.ListOptions) (*gastownv1alpha1.PolecatList, error)
	Create(ctx context.Context, polecat *gastownv1alpha1.Polecat) (*gastownv1alpha1.Polecat, error)
	Update(ctx context.Context, polecat *gastownv1alpha1.Polecat) (*gastownv1alpha1.Polecat, error)
	UpdateStatus(ctx context.Context, polecat *gastownv1alpha1.Polecat) (*gastownv1alpha1.Polecat, error)
	Delete(ctx context.Context, name string) error
}

// ConvoyClient provides typed operations for Convoy resources
type ConvoyClient interface {
	Get(ctx context.Context, name string) (*gastownv1alpha1.Convoy, error)
	List(ctx context.Context, opts metav1.ListOptions) (*gastownv1alpha1.ConvoyList, error)
	Create(ctx context.Context, convoy *gastownv1alpha1.Convoy) (*gastownv1alpha1.Convoy, error)
	Update(ctx context.Context, convoy *gastownv1alpha1.Convoy) (*gastownv1alpha1.Convoy, error)
	UpdateStatus(ctx context.Context, convoy *gastownv1alpha1.Convoy) (*gastownv1alpha1.Convoy, error)
	Delete(ctx context.Context, name string) error
}
