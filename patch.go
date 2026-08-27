package main

import (
	"io/ioutil"
	"strings"
)

func main() {
	content, _ := ioutil.ReadFile("internal/cli/review_next_transition.go")
	s := string(content)

	// Replace the StateReviewing block
	oldReviewing := `	case reviewtransaction.StateReviewing:
		if artifactErr != nil {
			return reviewStopTransition("captured_artifacts_unverifiable")
		}
		if len(artifacts) != len(selectedLenses) {
			return reviewMissingCaptureTransition(binding, selectedLenses, artifacts, input.CaptureContext, input.RuntimeAgent)
		}
		if input.ProviderRole == reviewerprovider.RoleRefuter {
			return reviewProviderRoleTransition("provider_refuter_required", binding, input.ProviderRole, input.RuntimeAgent, nil)
		}
		return reviewStopTransition("manual_intervention_required")`

	newReviewing := `	case reviewtransaction.StateReviewing:
		if artifactErr != nil {
			return reviewStopTransition("captured_artifacts_unverifiable")
		}
		admitted := make(map[int]bool, len(selectedLenses))
		unachievedAttempts := make(map[int]int, len(selectedLenses))
		completedArtifacts := make([]ReviewTransitionArtifact, 0, len(selectedLenses))
		for _, artifact := range artifacts {
			if isAdmittedArtifact(artifact) {
				admitted[artifact.SelectedOrder] = true
				completedArtifacts = append(completedArtifacts, artifact)
			} else if isUnachievableArtifact(artifact) {
				unachievedAttempts[artifact.SelectedOrder]++
			} else {
				return reviewStopTransition("captured_artifacts_unverifiable")
			}
		}
		for order := range selectedLenses {
			if !admitted[order] && unachievedAttempts[order] >= maxReviewerAttemptsPerSlot {
				return reviewStopTransition("unachievable_reviewer_attempt")
			}
		}
		if len(completedArtifacts) != len(selectedLenses) {
			return reviewMissingCaptureTransition(binding, selectedLenses, artifacts, input.CaptureContext, input.RuntimeAgent)
		}
		if input.ProviderRole == reviewerprovider.RoleRefuter {
			return reviewProviderRoleTransition("provider_refuter_required", binding, input.ProviderRole, input.RuntimeAgent, nil)
		}
		return reviewStopTransition("manual_intervention_required")`
	
	s = strings.Replace(s, oldReviewing, newReviewing, 1)

	// Replace reviewMissingCaptureTransition logic
	oldMissing := `func reviewMissingCaptureTransition(binding ReviewTransitionBinding, selectedLenses []string, artifacts []ReviewTransitionArtifact, context *reviewCaptureContext, runtime ...model.AgentID) ReviewNextTransition {
	providerRuntime := model.AgentID("")
	if len(runtime) > 0 && (reviewProviderCaptureRuntime(runtime[0]) || reviewProviderHostRelayMaterializeRuntime(runtime[0])) {
		providerRuntime = runtime[0]
	}
	captured := make(map[int]bool, len(artifacts))
	for _, artifact := range artifacts {
		captured[artifact.SelectedOrder] = true
	}
	inputs := make([]ReviewTransitionInput, 0)
	for order, lens := range selectedLenses {
		if !captured[order] {
			inputs = append(inputs, reviewCaptureInput(binding, lens, order, context, providerRuntime))
		}
	}
	if len(inputs) == 0 {
		return reviewStopTransition("captured_result_selection_unavailable")
	}
	return reviewCollectTransition("reviewer_results_required", inputs...)
}`

	newMissing := `func isAdmittedArtifact(artifact ReviewTransitionArtifact) bool {
	return artifact.AdmissionDecision == reviewtransaction.ArtifactAdmissionCompleted || artifact.AdmissionDecision == ""
}

func isUnachievableArtifact(artifact ReviewTransitionArtifact) bool {
	return artifact.AdmissionDecision == reviewtransaction.ArtifactAdmissionUnachievable
}

func reviewMissingCaptureTransition(binding ReviewTransitionBinding, selectedLenses []string, artifacts []ReviewTransitionArtifact, context *reviewCaptureContext, runtime ...model.AgentID) ReviewNextTransition {
	providerRuntime := model.AgentID("")
	if len(runtime) > 0 && (reviewProviderCaptureRuntime(runtime[0]) || reviewProviderHostRelayMaterializeRuntime(runtime[0])) {
		providerRuntime = runtime[0]
	}
	admitted := make(map[int]bool, len(selectedLenses))
	unachievedAttempts := make(map[int]int, len(selectedLenses))
	for _, artifact := range artifacts {
		if isAdmittedArtifact(artifact) {
			admitted[artifact.SelectedOrder] = true
		} else if isUnachievableArtifact(artifact) {
			unachievedAttempts[artifact.SelectedOrder]++
		} else {
			return reviewStopTransition("captured_artifacts_unverifiable")
		}
	}
	for order := range selectedLenses {
		if !admitted[order] && unachievedAttempts[order] >= maxReviewerAttemptsPerSlot {
			return reviewStopTransition("unachievable_reviewer_attempt")
		}
	}
	inputs := make([]ReviewTransitionInput, 0)
	for order, lens := range selectedLenses {
		if !admitted[order] {
			inputs = append(inputs, reviewCaptureInput(binding, lens, order, context, providerRuntime))
		}
	}
	if len(inputs) == 0 {
		return reviewStopTransition("captured_result_selection_unavailable")
	}
	return reviewCollectTransition("reviewer_results_required", inputs...)
}`

	s = strings.Replace(s, oldMissing, newMissing, 1)

	ioutil.WriteFile("internal/cli/review_next_transition.go", []byte(s), 0644)
}
