// Copyright 2021 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package java

import (
	"fmt"

	"android/soong/android"
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// Contains code that is common to both platform_bootclasspath and bootclasspath_fragment.

// addDependencyOntoApexVariants adds dependencies onto the appropriate apex specific variants of
// the module as specified in the ApexVariantReference list.
func addDependencyOntoApexVariants(ctx android.BottomUpMutatorContext, propertyName string, refs []ApexVariantReference, tag blueprint.DependencyTag) {
	for i, ref := range refs {
		apex := proptools.StringDefault(ref.Apex, "platform")

		if ref.Module == nil {
			ctx.PropertyErrorf(propertyName, "missing module name at position %d", i)
			continue
		}
		name := proptools.String(ref.Module)

		addDependencyOntoApexModulePair(ctx, apex, name, tag)
	}
}

// addDependencyOntoApexModulePair adds a dependency onto the specified APEX specific variant or the
// specified module.
//
// If apex="platform" then this adds a dependency onto the platform variant of the module. This adds
// dependencies onto the prebuilt and source modules with the specified name, depending on which
// ones are available. Visiting must use isActiveModule to select the preferred module when both
// source and prebuilt modules are available.
func addDependencyOntoApexModulePair(ctx android.BottomUpMutatorContext, apex string, name string, tag blueprint.DependencyTag) {
	var variations []blueprint.Variation
	if apex != "platform" {
		// Pick the correct apex variant.
		variations = []blueprint.Variation{
			{Mutator: "apex", Variation: apex},
		}
	}

	addedDep := false
	if ctx.OtherModuleDependencyVariantExists(variations, name) {
		ctx.AddFarVariationDependencies(variations, tag, name)
		addedDep = true
	}

	// Add a dependency on the prebuilt module if it exists.
	prebuiltName := android.PrebuiltNameFromSource(name)
	if ctx.OtherModuleDependencyVariantExists(variations, prebuiltName) {
		ctx.AddVariationDependencies(variations, tag, prebuiltName)
		addedDep = true
	}

	// If no appropriate variant existing for this, so no dependency could be added, then it is an
	// error, unless missing dependencies are allowed. The simplest way to handle that is to add a
	// dependency that will not be satisfied and the default behavior will handle it.
	if !addedDep {
		// Add dependency on the unprefixed (i.e. source or renamed prebuilt) module which we know does
		// not exist. The resulting error message will contain useful information about the available
		// variants.
		reportMissingVariationDependency(ctx, variations, name)

		// Add dependency on the missing prefixed prebuilt variant too if a module with that name exists
		// so that information about its available variants will be reported too.
		if ctx.OtherModuleExists(prebuiltName) {
			reportMissingVariationDependency(ctx, variations, prebuiltName)
		}
	}
}

// reportMissingVariationDependency intentionally adds a dependency on a missing variation in order
// to generate an appropriate error message with information about the available variations.
func reportMissingVariationDependency(ctx android.BottomUpMutatorContext, variations []blueprint.Variation, name string) {
	modules := ctx.AddFarVariationDependencies(variations, nil, name)
	if len(modules) != 1 {
		panic(fmt.Errorf("Internal Error: expected one module, found %d", len(modules)))
		return
	}
	if modules[0] != nil {
		panic(fmt.Errorf("Internal Error: expected module to be missing but was found: %q", modules[0]))
		return
	}
}

// ApexVariantReference specifies a particular apex variant of a module.
type ApexVariantReference struct {
	// The name of the module apex variant, i.e. the apex containing the module variant.
	//
	// If this is not specified then it defaults to "platform" which will cause a dependency to be
	// added to the module's platform variant.
	Apex *string

	// The name of the module.
	Module *string
}

// BootclasspathFragmentsDepsProperties contains properties related to dependencies onto fragments.
type BootclasspathFragmentsDepsProperties struct {
	// The names of the bootclasspath_fragment modules that form part of this module.
	Fragments []ApexVariantReference
}

// addDependenciesOntoFragments adds dependencies to the fragments specified in this properties
// structure.
func (p *BootclasspathFragmentsDepsProperties) addDependenciesOntoFragments(ctx android.BottomUpMutatorContext) {
	addDependencyOntoApexVariants(ctx, "fragments", p.Fragments, bootclasspathFragmentDepTag)
}

// bootclasspathDependencyTag defines dependencies from/to bootclasspath_fragment,
// prebuilt_bootclasspath_fragment and platform_bootclasspath onto either source or prebuilt
// modules.
type bootclasspathDependencyTag struct {
	blueprint.BaseDependencyTag

	name string
}

func (t bootclasspathDependencyTag) ExcludeFromVisibilityEnforcement() {
}

// Dependencies that use the bootclasspathDependencyTag instances are only added after all the
// visibility checking has been done so this has no functional effect. However, it does make it
// clear that visibility is not being enforced on these tags.
var _ android.ExcludeFromVisibilityEnforcementTag = bootclasspathDependencyTag{}

// The tag used for dependencies onto bootclasspath_fragments.
var bootclasspathFragmentDepTag = bootclasspathDependencyTag{name: "fragment"}
