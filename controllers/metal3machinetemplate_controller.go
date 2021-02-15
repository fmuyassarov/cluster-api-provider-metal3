/*
Copyright 2019 The Kubernetes Authors.
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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	capm3 "github.com/metal3-io/cluster-api-provider-metal3/api/v1alpha4"
	"github.com/metal3-io/cluster-api-provider-metal3/baremetal"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	templateControllerName = "Metal3MachineTemplate-controller"
)

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3machinestemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3machinestemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3machines,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3machines/status,verbs=get

// Metal3MachineTemplateReconciler reconciles a Metal3MachineTemplate object
type Metal3MachineTemplateReconciler struct {
	Client         client.Client
	ManagerFactory baremetal.ManagerFactoryInterface
	Log            logr.Logger
}

// Reconcile handles Metal3MachineTemplate events
func (r *Metal3MachineTemplateReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, rerr error) {
	ctx := context.Background()
	m3templateLog := r.Log.WithValues(templateControllerName).WithValues("metal3-machine-template", req.NamespacedName)

	// Fetch the Metal3MachineTemplate instance.
	metal3MachineTemplate := &capm3.Metal3MachineTemplate{}

	if err := r.Client.Get(ctx, req.NamespacedName, metal3MachineTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, errors.Wrap(err, "unable to fetch Metal3MachineTemplate")
	}

	helper, err := patch.NewHelper(metal3MachineTemplate, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to init patch helper")
	}

	// Always patch metal3MachineTemplate exiting this function so we can persist any metal3MachineTemplate changes.
	defer func() {
		err := helper.Patch(ctx, metal3MachineTemplate)
		if err != nil {
			m3templateLog.Info("failed to Patch Metal3MachineTemplate")
		}
	}()

	// Fetch the Metal3Machine instance.
	m3machine := &capm3.Metal3Machine{}

	// Fetch the Metal3MachineList
	m3machinelist := &capm3.Metal3MachineList{}

	// Create a helper for managing a Metal3MachineTemplate.
	templateMgr, err := r.ManagerFactory.NewMachineTemplateManager(metal3MachineTemplate, m3machine, m3machinelist, m3templateLog)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to create helper for managing the templateMgr")
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, templateMgr)
}

func (r *Metal3MachineTemplateReconciler) reconcileNormal(ctx context.Context,
	templateMgr baremetal.TemplateManagerInterface,
) (ctrl.Result, error) {

	// Find the Metal3Machines with clonedFromName annotation referencing
	// to the same Metal3MachineTemplate
	machinesWithAnnotation, err := templateMgr.FindMachine(ctx)
	if err != nil {
		r.Log.Error(err, "failed to list Metal3Machines with clonedFromName annotation")
		return ctrl.Result{}, err
	}

	if err := templateMgr.SyncDisableAutomatedClean(machinesWithAnnotation); err != nil {
		r.Log.Error(err, "failed to set DisableAutomatedClean annotation on metal3Machines")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager will add watches for this controller
func (r *Metal3MachineTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&capm3.Metal3MachineTemplate{}).
		Complete(r)
}
