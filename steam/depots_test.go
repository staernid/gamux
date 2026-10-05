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

func TestFetchAppBranchManifestGIDs(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockJSON := `{
		"status": "success",
		"data": {
			"993090": {
				"depots": {
					"993091": {
						"name": "Lossless Scaling Content",
						"config": {
							"oslist": "windows"
						},
						"manifests": {
							"public": {
								"download": "57169184",
								"gid": "2549887697121492489",
								"size": "183809856"
							},
							"lsfg-vk": {
								"download": "2100784",
								"gid": "335002223878656166",
								"size": "9664532"
							},
							"legacy_3.1": {
								"download": "56445712",
								"gid": "9191341654061558981",
								"size": "181296520"
							}
						}
					},
					"993092": {
						"name": "Lossless Scaling China Content",
						"config": {
							"realm": "steamchina"
						},
						"manifests": {
							"public": {
								"gid": "3384896674095978390"
							}
						}
					},
					"branches": {}
				}
			}
		}
	}`

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "993090") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockJSON)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("not found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)

	t.Run("public branch excludes steamchina", func(t *testing.T) {
		gids, err := FetchAppBranchManifestGIDs(ctx, 993090, "public")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gids[993091] != "2549887697121492489" {
			t.Errorf("expected public gid 2549887697121492489, got %q", gids[993091])
		}
		if _, exists := gids[993092]; exists {
			t.Errorf("expected steamchina depot 993092 to be excluded, got %q", gids[993092])
		}
	})

	t.Run("lsfg-vk branch", func(t *testing.T) {
		gids, err := FetchAppBranchManifestGIDs(ctx, 993090, "lsfg-vk")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gids[993091] != "335002223878656166" {
			t.Errorf("expected lsfg-vk gid 335002223878656166, got %q", gids[993091])
		}
		if _, exists := gids[993092]; exists {
			t.Errorf("expected steamchina depot 993092 to be excluded, got %q", gids[993092])
		}
	})

	t.Run("fallback to public for unknown branch", func(t *testing.T) {
		gids, err := FetchAppBranchManifestGIDs(ctx, 993090, "nonexistent-branch")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gids[993091] != "2549887697121492489" {
			t.Errorf("expected fallback to public gid 2549887697121492489, got %q", gids[993091])
		}
		if _, exists := gids[993092]; exists {
			t.Errorf("expected steamchina depot 993092 to be excluded, got %q", gids[993092])
		}
	})
}

func TestFetchAppDepotMetadata_EnrichmentAndSynthesis(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockFactorioJSON := `{
		"status": "success",
		"data": {
			"427520": {
				"common": {
					"name": "Factorio",
					"type": "Game"
				},
				"depots": {
					"427521": {
						"config": {
							"osarch": "64",
							"oslist": "windows"
						},
						"manifests": {
							"public": {
								"download": "1560048000",
								"gid": "4175659602074733495",
								"size": "2249816030"
							}
						}
					},
					"427523": {
						"config": {
							"osarch": "64",
							"oslist": "linux"
						},
						"manifests": {
							"public": {
								"download": "1621678976",
								"gid": "2157676814121005612",
								"size": "2593227559"
							}
						}
					},
					"645391": {
						"dlcappid": "645390",
						"config": {
							"osarch": "64",
							"oslist": "linux"
						},
						"manifests": {
							"public": {
								"download": "4276014000",
								"gid": "5217534433594943675",
								"size": "4844866835"
							}
						}
					}
				}
			}
		}
	}`

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "427520") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockFactorioJSON)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("not found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)

	depots, err := FetchAppDepotMetadata(ctx, 427520)
	if err != nil {
		t.Fatalf("FetchAppDepotMetadata failed: %v", err)
	}

	// 1. Verify depot 427521 (Windows 64-bit)
	d427521, ok := depots[427521]
	if !ok {
		t.Fatalf("expected depot 427521 to be present")
	}
	if d427521.Size != 2249816030 {
		t.Errorf("expected size 2249816030, got %d", d427521.Size)
	}
	if d427521.DownloadSize != 1560048000 {
		t.Errorf("expected download size 1560048000, got %d", d427521.DownloadSize)
	}
	if d427521.OSArch != "64" {
		t.Errorf("expected osarch '64', got %q", d427521.OSArch)
	}
	if !strings.Contains(d427521.Name, "Windows Binaries (64-bit)") {
		t.Errorf("expected synthesized name to include 'Windows Binaries (64-bit)', got %q", d427521.Name)
	}

	// 2. Verify depot 645391 (DLC depot)
	d645391, ok := depots[645391]
	if !ok {
		t.Fatalf("expected depot 645391 to be present")
	}
	if !d645391.IsDLC {
		t.Errorf("expected depot 645391 to be marked as DLC")
	}
	if d645391.DLCAppID != 645390 {
		t.Errorf("expected DLCAppID 645390, got %d", d645391.DLCAppID)
	}
	if d645391.Size != 4844866835 {
		t.Errorf("expected size 4844866835, got %d", d645391.Size)
	}
	if !strings.Contains(d645391.Name, "DLC #645390") {
		t.Errorf("expected name to reference DLC #645390, got %q", d645391.Name)
	}
}

func TestFetchAppDepotMetadata_DLCParentResolution(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockSpaceAgeJSON := `{
		"status": "success",
		"data": {
			"645390": {
				"common": {
					"name": "Factorio: Space Age",
					"parent": "427520",
					"type": "DLC"
				},
				"depots": {
					"645391": {
						"config": {
							"osarch": "64",
							"oslist": "linux"
						}
					}
				}
			}
		}
	}`

	mockFactorioParentJSON := `{
		"status": "success",
		"data": {
			"427520": {
				"common": {
					"name": "Factorio",
					"type": "Game"
				},
				"depots": {
					"427521": {
						"config": { "oslist": "windows" },
						"manifests": { "public": { "size": "1000", "gid": "111" } }
					},
					"645391": {
						"dlcappid": "645390",
						"config": {
							"osarch": "64",
							"oslist": "linux"
						},
						"manifests": {
							"public": {
								"size": "5000000000",
								"download": "4000000000",
								"gid": "999"
							}
						}
					}
				}
			}
		}
	}`

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "645390") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockSpaceAgeJSON)),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(req.URL.Path, "427520") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockFactorioParentJSON)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("not found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)

	depots, err := FetchAppDepotMetadata(ctx, 645390)
	if err != nil {
		t.Fatalf("FetchAppDepotMetadata failed: %v", err)
	}

	d645391, ok := depots[645391]
	if !ok {
		t.Fatalf("expected depot 645391 to be resolved from parent")
	}
	if d645391.Size != 5000000000 {
		t.Errorf("expected inherited size 5000000000, got %d", d645391.Size)
	}
	if !strings.Contains(d645391.Name, "Factorio: Space Age") {
		t.Errorf("expected synthesized name to include DLC name 'Factorio: Space Age', got %q", d645391.Name)
	}
}

func TestDiscoverDLCs_Rimworld(t *testing.T) {
	dlcs, err := DiscoverDLCs(context.Background(), "294100")
	t.Logf("DiscoverDLCs Rimworld err: %v, count: %d", err, len(dlcs))
	for _, d := range dlcs {
		t.Logf("DLC: AppID=%s, Name=%q", d.AppID, d.Name)
	}
}

