// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

//go:build integration

// Package provisioner_test hosts shared helpers for mock-based integration tests
// that use httptest.NewServer to stub OCI API responses.
package provisioner_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
)

// route is a method+path key into the responses map.
type route struct {
	method, path string
}

// canned is a stubbed HTTP response.
type canned struct {
	status int
	body   string
}

// newTestDispatcher starts an httptest server that returns canned responses
// for the given routes. Any unregistered request fails the test.
// Returns the server URL, to be set as client.Host.
func newTestDispatcher(t *testing.T, responses map[route]canned) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := responses[route{r.Method, r.URL.Path}]
		if !ok {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(c.status)
		fmt.Fprint(w, c.body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// testKeyOnce caches a single RSA key for all tests. Generating a 2048-bit key
// takes ~100ms; generating one per test (60+ tests) adds seconds of pure waste.
// The test server never validates signatures, so one shared key is safe.
var (
	testKeyOnce sync.Once
	testKeyPEM  string
)

func getTestKeyPEM(t *testing.T) string {
	t.Helper()
	testKeyOnce.Do(func() {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("failed to generate test RSA key: %v", err)
		}
		testKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}))
	})
	return testKeyPEM
}

// fakeOCIConfigProvider returns a ConfigurationProvider suitable for tests.
// It supplies a dummy RSA key so the SDK can sign requests — the test server
// ignores the signature, so the key content doesn't matter.
func fakeOCIConfigProvider(t *testing.T) common.ConfigurationProvider {
	t.Helper()
	return common.NewRawConfigurationProvider(
		"ocid1.tenancy.oc1..test",
		"ocid1.user.oc1..test",
		"us-chicago-1",
		"aa:bb:cc:dd:ee:ff:11:22:33:44:55:66:77:88:99:00",
		getTestKeyPEM(t),
		nil,
	)
}

// noECRetryPolicyForTests is applied to test clients so that 404 responses are
// returned immediately instead of triggering the SDK's eventual-consistency retry
// loop (which retries for up to 4 minutes after any write operation).
var noECRetryPolicyForTests = common.DefaultRetryPolicyWithoutEventualConsistency()

// applyTestRetryPolicy sets the no-EC-retry policy on any OCI client that has
// a SetCustomClientConfiguration method. Callers pass their concrete client.
func applyTestRetryPolicy(c interface {
	SetCustomClientConfiguration(common.CustomClientConfiguration)
}) {
	c.SetCustomClientConfiguration(common.CustomClientConfiguration{RetryPolicy: &noECRetryPolicyForTests})
}
