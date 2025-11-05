package stdlib

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/typechecker"
)

// TestRegistrySyncWithTypechecker ensures that the stdlib registry is in sync
// with the actual typechecker implementation. This test will fail if:
// - A function exists in the typechecker but not in the registry
// - A function exists in the registry but not in the typechecker
// - The number of functions in a namespace differs
//
// This test is critical for CI to ensure the registry doesn't drift from reality.
func TestRegistrySyncWithTypechecker(t *testing.T) {
	// Get all namespaces from both sources
	registryNamespaces := GetNamespaces()
	typecheckerNamespaces := make([]string, 0)
	for namespace := range typechecker.StdlibFunctions {
		typecheckerNamespaces = append(typecheckerNamespaces, namespace)
	}

	// Sort typechecker namespaces for comparison
	for i := 0; i < len(typecheckerNamespaces); i++ {
		for j := i + 1; j < len(typecheckerNamespaces); j++ {
			if typecheckerNamespaces[i] > typecheckerNamespaces[j] {
				typecheckerNamespaces[i], typecheckerNamespaces[j] = typecheckerNamespaces[j], typecheckerNamespaces[i]
			}
		}
	}

	// Verify namespace counts match
	if len(registryNamespaces) != len(typecheckerNamespaces) {
		t.Errorf("Registry has %d namespaces, typechecker has %d namespaces",
			len(registryNamespaces), len(typecheckerNamespaces))
		t.Errorf("Registry namespaces: %v", registryNamespaces)
		t.Errorf("Typechecker namespaces: %v", typecheckerNamespaces)
	}

	// Verify each namespace exists in both
	for _, namespace := range registryNamespaces {
		found := false
		for _, tcNamespace := range typecheckerNamespaces {
			if namespace == tcNamespace {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Namespace %s exists in registry but not in typechecker", namespace)
		}
	}

	for _, namespace := range typecheckerNamespaces {
		found := false
		for _, regNamespace := range registryNamespaces {
			if namespace == regNamespace {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Namespace %s exists in typechecker but not in registry", namespace)
		}
	}

	// For each namespace, verify function counts match
	for _, namespace := range registryNamespaces {
		registryFuncs := GetFunctions(namespace)
		typecheckerFuncs, ok := typechecker.StdlibFunctions[namespace]

		if !ok {
			t.Errorf("Namespace %s exists in registry but not in typechecker", namespace)
			continue
		}

		if len(registryFuncs) != len(typecheckerFuncs) {
			t.Errorf("Namespace %s: registry has %d functions, typechecker has %d functions",
				namespace, len(registryFuncs), len(typecheckerFuncs))

			// List the functions for debugging
			t.Logf("Registry functions in %s:", namespace)
			for _, fn := range registryFuncs {
				t.Logf("  - %s", fn.Name)
			}
			t.Logf("Typechecker functions in %s:", namespace)
			for name := range typecheckerFuncs {
				t.Logf("  - %s", name)
			}
		}

		// Verify each function exists in both
		for _, regFunc := range registryFuncs {
			if _, exists := typecheckerFuncs[regFunc.Name]; !exists {
				t.Errorf("Function %s.%s exists in registry but not in typechecker",
					namespace, regFunc.Name)
			}
		}

		for tcFuncName := range typecheckerFuncs {
			found := false
			for _, regFunc := range registryFuncs {
				if regFunc.Name == tcFuncName {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Function %s.%s exists in typechecker but not in registry",
					namespace, tcFuncName)
			}
		}
	}
}

// TestRegistrySignaturesMatchTypechecker verifies that function signatures
// in the registry accurately reflect the typechecker implementation
func TestRegistrySignaturesMatchTypechecker(t *testing.T) {
	for namespace, funcs := range StdlibRegistry {
		typecheckerFuncs, ok := typechecker.StdlibFunctions[namespace]
		if !ok {
			t.Errorf("Namespace %s exists in registry but not in typechecker", namespace)
			continue
		}

		for _, regFunc := range funcs {
			tcFunc, exists := typecheckerFuncs[regFunc.Name]
			if !exists {
				t.Errorf("Function %s.%s exists in registry but not in typechecker",
					namespace, regFunc.Name)
				continue
			}

			// Verify parameter count
			expectedParamCount := len(tcFunc.Parameters)
			actualParamCount := countParameters(regFunc.Signature)

			if actualParamCount != expectedParamCount {
				t.Errorf("Function %s.%s: registry signature has %d parameters, typechecker has %d",
					namespace, regFunc.Name, actualParamCount, expectedParamCount)
				t.Logf("  Registry signature: %s", regFunc.Signature)
			}

			// Verify return type nullability
			expectedNullable := tcFunc.ReturnType.IsNullable()
			actualNullable := isReturnTypeNullable(regFunc.Signature)

			if actualNullable != expectedNullable {
				t.Errorf("Function %s.%s: return type nullability mismatch",
					namespace, regFunc.Name)
				t.Logf("  Registry signature: %s", regFunc.Signature)
				t.Logf("  Expected nullable: %v, got: %v", expectedNullable, actualNullable)
			}
		}
	}
}

// countParameters counts the number of parameters in a function signature
// by counting commas in the parameter list (handles 0, 1, 2+ parameters)
func countParameters(signature string) int {
	// Find the parameter list (between '(' and ')')
	parenStart := -1
	parenEnd := -1

	for i, ch := range signature {
		if ch == '(' {
			parenStart = i
		} else if ch == ')' {
			parenEnd = i
			break
		}
	}

	if parenStart == -1 || parenEnd == -1 || parenEnd <= parenStart+1 {
		// No parameters: func()
		return 0
	}

	// Count parameters by counting commas + 1
	paramList := signature[parenStart+1 : parenEnd]
	commaCount := 0
	for _, ch := range paramList {
		if ch == ',' {
			commaCount++
		}
	}

	return commaCount + 1
}

// isReturnTypeNullable checks if the return type in a signature is nullable
// by looking for '?' after the '->' arrow
func isReturnTypeNullable(signature string) bool {
	// Find the return type (after '->')
	arrowIndex := -1
	for i := 0; i < len(signature)-1; i++ {
		if signature[i] == '-' && signature[i+1] == '>' {
			arrowIndex = i
			break
		}
	}

	if arrowIndex == -1 {
		return false
	}

	// Check if there's a '?' after the arrow
	for i := arrowIndex + 2; i < len(signature); i++ {
		if signature[i] == '?' {
			return true
		}
		if signature[i] == '!' {
			return false
		}
	}

	return false
}

// TestRegistryUpdateInstructions provides clear instructions when the registry is out of sync
func TestRegistryUpdateInstructions(t *testing.T) {
	// This test always passes, but logs helpful instructions if sync issues are detected
	var syncIssues []string

	for namespace := range typechecker.StdlibFunctions {
		if GetFunctions(namespace) == nil {
			syncIssues = append(syncIssues, "Missing namespace: "+namespace)
		}
	}

	for namespace, funcs := range typechecker.StdlibFunctions {
		registryFuncs := GetFunctions(namespace)
		if registryFuncs == nil {
			continue
		}

		for funcName := range funcs {
			found := false
			for _, regFunc := range registryFuncs {
				if regFunc.Name == funcName {
					found = true
					break
				}
			}
			if !found {
				syncIssues = append(syncIssues, "Missing function: "+namespace+"."+funcName)
			}
		}
	}

	if len(syncIssues) > 0 {
		t.Log("======================================================================")
		t.Log("REGISTRY OUT OF SYNC!")
		t.Log("======================================================================")
		t.Log("")
		t.Log("The stdlib registry is out of sync with the typechecker.")
		t.Log("Please update internal/compiler/stdlib/registry.go to include:")
		t.Log("")
		for _, issue := range syncIssues {
			t.Log("  - " + issue)
		}
		t.Log("")
		t.Log("Steps to fix:")
		t.Log("1. Open internal/compiler/stdlib/registry.go")
		t.Log("2. Add the missing namespaces/functions to StdlibRegistry")
		t.Log("3. Include proper signatures and descriptions")
		t.Log("4. Run tests again to verify sync")
		t.Log("")
		t.Log("======================================================================")
	}
}
