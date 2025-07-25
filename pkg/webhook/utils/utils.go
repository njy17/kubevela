/*
 Copyright 2021. The KubeVela Authors.

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

package utils

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kubevela/pkg/cue/cuex"

	"cuelang.org/go/cue/cuecontext"
	cueErrors "cuelang.org/go/cue/errors"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/oam-dev/kubevela/apis/core.oam.dev/v1beta1"
	"github.com/oam-dev/kubevela/pkg/controller/core.oam.dev/v1beta1/core"
)

// ContextRegex to match '**: reference "context" not found'
var ContextRegex = `^.+:\sreference\s\"context\"\snot\sfound$`

// ValidateDefinitionRevision validate whether definition will modify the immutable object definitionRevision
func ValidateDefinitionRevision(ctx context.Context, cli client.Client, def runtime.Object, defRevNamespacedName types.NamespacedName) error {
	if errs := validation.IsQualifiedName(defRevNamespacedName.Name); len(errs) != 0 {
		return errors.Errorf("invalid definitionRevision name %s:%s", defRevNamespacedName.Name, strings.Join(errs, ","))
	}
	defRev := new(v1beta1.DefinitionRevision)
	if err := cli.Get(ctx, defRevNamespacedName, defRev); err != nil {
		return client.IgnoreNotFound(err)
	}

	newRev, _, err := core.GatherRevisionInfo(def)
	if err != nil {
		return err
	}
	if defRev.Spec.RevisionHash != newRev.Spec.RevisionHash {
		return errors.New("the definition's spec is different with existing definitionRevision's spec")
	}
	if !core.DeepEqualDefRevision(defRev, newRev) {
		return errors.New("the definition's spec is different with existing definitionRevision's spec")
	}
	return nil
}

// ValidateCueTemplate validate cueTemplate
func ValidateCueTemplate(cueTemplate string) error {

	val := cuecontext.New().CompileString(cueTemplate)
	if e := checkError(val.Err()); e != nil {
		return e
	}

	err := val.Validate()
	return checkError(err)
}

// ValidateCuexTemplate validate cueTemplate with CueX for types utilising it
func ValidateCuexTemplate(ctx context.Context, cueTemplate string) error {
	val, err := cuex.DefaultCompiler.Get().CompileStringWithOptions(ctx, cueTemplate)
	if err != nil {
		return err
	}
	if e := checkError(val.Err()); e != nil {
		return e
	}
	err = val.Validate()
	return checkError(err)
}

func checkError(err error) error {
	re := regexp.MustCompile(ContextRegex)
	if err != nil {
		// ignore context not found error
		for _, e := range cueErrors.Errors(err) {
			if !re.MatchString(e.Error()) {
				return cueErrors.New(e.Error())
			}
		}
	}
	return nil
}

// ValidateSemanticVersion validates if a Definition's version includes all of
// major,minor & patch version values.
func ValidateSemanticVersion(version string) error {
	if version != "" {
		versionParts := strings.Split(version, ".")
		if len(versionParts) != 3 {
			return errors.New("Not a valid version")
		}

		for _, versionPart := range versionParts {
			if _, err := strconv.Atoi(versionPart); err != nil {
				return errors.New("Not a valid version")
			}
		}
	}
	return nil
}

// ValidateMultipleDefVersionsNotPresent validates that both Name Annotation Revision and Spec.Version are not present
func ValidateMultipleDefVersionsNotPresent(version, revisionName, objectType string) error {
	if version != "" && revisionName != "" {
		return fmt.Errorf("%s has both spec.version and revision name annotation. Only one can be present", objectType)
	}
	return nil
}
