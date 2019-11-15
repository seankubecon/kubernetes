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

package foo

import (
	"fmt"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"
)

const defaultFilename = "default.yaml"

var (
	fooLong = templates.LongDesc(i18n.T(`
This is the foo command long description.
`))

	fooExample = templates.Examples(i18n.T(`
		# Foo command example
		kubectl foo --count 3 --filename foo-resource.yaml
`))
)

// FooOptions are the knobs available for the "foo" command.
type FooOptions struct {
	Count            int
	FilenameOptions  resource.FilenameOptions
	namespace        string
	enforceNamespace bool
	PrintFlags       *genericclioptions.PrintFlags
}

// NewCmdFoo a new Cobra command encasulating the "foo" command.
func NewCmdFoo(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	o := &FooOptions{
		PrintFlags: genericclioptions.NewPrintFlags("created").WithDefaultOutput("name"),
	}

	cmd := &cobra.Command{
		Use: "foo [--count=COUNT] --filename=FILENAME",
		DisableFlagsInUseLine: true,
		Short:   i18n.T("Foo short description"),
		Long:    fooLong,
		Example: fooExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(f, args))
			cmdutil.CheckErr(o.Validate())
			cmdutil.CheckErr(o.RunFoo(f, ioStreams))
		},
	}

	o.PrintFlags.AddFlags(cmd)

	cmd.Flags().IntVarP(&o.Count, "count", "c", o.Count, "Usage for count flag.")
	cmdutil.AddFilenameOptionFlags(cmd, &o.FilenameOptions, "")

	return cmd
}

// Complete fills in all the FooOptions fields, including defaults.
func (o *FooOptions) Complete(f cmdutil.Factory, args []string) error {

	var err error
	o.namespace, o.enforceNamespace, err = f.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	return nil
}

// Validate ensures all FooOptions fields are valid.
func (o *FooOptions) Validate() error {
	if o.Count < 0 {
		return fmt.Errorf("Count is negative")
	}

	return nil
}

// RunFoo executes the foo command.
func (o *FooOptions) RunFoo(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) error {

	// The resource builder is used to retrieve and decode the resource from the
	// local filesystem (e.g from a YAML file). This resource builder will specify the
	// filename using the FilenameParam() method, reading the filename into the
	// o.FilenameOptions variable. The Scheme specifies all the Group/Versions that
	// this kubectl knows. Once the Group/Version/Kind specified in the YAML is matched
	// to a GVK from the Scheme, then it can be decoded into an object of type GVK.
	// An example of a GVK is apps/v1/Deployment. If the group is missing, it is in the
	// "core" group. An example of this is core/v1/Pod. After the "Do" method creates
	// the result, we call r.Visit() to iterate through the resources (there can
	// be more than one).
	r := f.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ContinueOnError().
		NamespaceParam(o.namespace).DefaultNamespace().
		FilenameParam(o.enforceNamespace, &o.FilenameOptions).
		Flatten().
		Do()
	err := r.Err()
	if err != nil {
		return err
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return err
	}

	// Iterate through the result objects (in the resource.Info).
	var obj runtime.Object
	err = r.Visit(func(info *resource.Info, err error) error {
		if err == nil {
			obj = info.Object

			// Create the resource helper. The parameters are a RESTMapping
			// (essentially a GVK), and a RESTClient (created by the
			// factory.ClientForMapping() method).
			mapping := info.ResourceMapping()
			client, err := f.ClientForMapping(mapping)
			if err != nil {
				return err
			}
			helper := resource.NewHelper(client, mapping)

			// Using the resource helper, create the decoded object on the APIServer.
			_, err = helper.Create(info.Namespace, true, obj, &metav1.CreateOptions{})
			if err != nil {
				return err
			}
			printer.PrintObj(obj, ioStreams.Out)
		}
		return nil
	})

	return nil
}
