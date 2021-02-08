package main

import (
	"github.com/EpiK-Protocol/go-epik/testplans/lotus-soup/paych"
	"github.com/EpiK-Protocol/go-epik/testplans/lotus-soup/rfwp"
	"github.com/EpiK-Protocol/go-epik/testplans/lotus-soup/testkit"
	"github.com/testground/sdk-go/run"
)

var cases = map[string]interface{}{
	"deals-e2e":                     testkit.WrapTestEnvironment(dealsE2E),
	"recovery-failed-windowed-post": testkit.WrapTestEnvironment(rfwp.RecoveryFromFailedWindowedPoStE2E),
	"deals-stress":                  testkit.WrapTestEnvironment(dealsStress),
	"drand-halting":                 testkit.WrapTestEnvironment(dealsE2E),
	"drand-outage":                  testkit.WrapTestEnvironment(dealsE2E),
	"paych-stress":                  testkit.WrapTestEnvironment(paych.Stress),
	"eco-vote":                      testkit.WrapTestEnvironment(ecoVote),
	"eco-retrieve-pledge":           testkit.WrapTestEnvironment(ecoRetrievePledge),
}

func main() {
	sanityCheck()

	run.InvokeMap(cases)
}
