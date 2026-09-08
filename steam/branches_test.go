package steam

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/util/testutil"
)

func TestFetchAppBranches(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	t.Run("invalid appID 0", func(t *testing.T) {
		_, err := FetchAppBranches(context.Background(), 0)
		if err == nil {
			t.Fatal("expected error for appID 0, got nil")
		}
	})

	t.Run("successful response with multiple branches", func(t *testing.T) {
		mockJSON := `{
			"status": "success",
			"data": {
				"646570": {
					"depots": {
						"branches": {
							"v1.0": {
								"buildid": "3495375",
								"description": "2019-23-01",
								"timebuildupdated": "1548268856",
								"timeupdated": "1562010477"
							},
							"public": {
								"buildid": "10180494",
								"timebuildupdated": "1671496024",
								"timeupdated": "1671497797"
							},
							"beta": {
								"buildid": "10174821",
								"description": "Beta branch for testing new content and risky fixes. Unstable",
								"timebuildupdated": "1671430290",
								"timeupdated": "1671497193"
							},
							"private_test": {
								"buildid": "999999",
								"description": "Internal test",
								"timeupdated": "1671497193",
								"pwdrequired": "1"
							}
						}
					}
				}
			}
		}`

		mockClient := &http.Client{
			Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if !strings.Contains(req.URL.Path, "646570") {
					t.Errorf("unexpected URL path: %s", req.URL.Path)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockJSON)),
					Header:     make(http.Header),
				}, nil
			}),
		}

		ctx := WithHTTPClient(context.Background(), mockClient)
		branches, err := FetchAppBranches(ctx, 646570)
		if err != nil {
			t.Fatalf("FetchAppBranches failed: %v", err)
		}

		if len(branches) != 4 {
			t.Fatalf("expected 4 branches, got %d", len(branches))
		}

		// Check public is first
		if branches[0].Name != "public" {
			t.Errorf("expected first branch to be 'public', got %q", branches[0].Name)
		}
		if branches[0].BuildID != "10180494" {
			t.Errorf("expected public buildid 10180494, got %q", branches[0].BuildID)
		}

		// Check beta is sorted alphabetically after public
		if branches[1].Name != "beta" {
			t.Errorf("expected second branch to be 'beta', got %q", branches[1].Name)
		}
		if branches[1].Description != "Beta branch for testing new content and risky fixes. Unstable" {
			t.Errorf("unexpected description: %s", branches[1].Description)
		}
		if branches[1].BuildID != "10174821" {
			t.Errorf("expected buildid 10174821, got %q", branches[1].BuildID)
		}
		if branches[1].PwdRequired {
			t.Error("expected beta to not require password")
		}

		// Check private_test has PwdRequired == true
		var foundPrivate bool
		for _, b := range branches {
			if b.Name == "private_test" {
				foundPrivate = true
				if !b.PwdRequired {
					t.Error("expected private_test to require password")
				}
			}
		}
		if !foundPrivate {
			t.Error("private_test branch not found")
		}
	})

	t.Run("fallback when no branches in response", func(t *testing.T) {
		mockJSON := `{
			"status": "success",
			"data": {
				"123": {
					"depots": {}
				}
			}
		}`

		mockClient := &http.Client{
			Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockJSON)),
					Header:     make(http.Header),
				}, nil
			}),
		}

		ctx := WithHTTPClient(context.Background(), mockClient)
		branches, err := FetchAppBranches(ctx, 123)
		if err != nil {
			t.Fatalf("FetchAppBranches failed: %v", err)
		}

		if len(branches) != 1 || branches[0].Name != "public" {
			t.Fatalf("expected fallback public branch, got: %+v", branches)
		}
	})
}
