package downloader

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/util/testutil"
)

// cdnReply is the scripted response of a fakeCDN for a single chunk request.
type cdnReply struct {
	status int
	body   []byte
}

// fakeCDN is an in-memory Steam CDN. Chunks are addressed by their (lowercase) chunk ID and
// served from any host; a script hook can override the reply per host/chunk/hit to simulate
// outages, corruption, and mid-download failures. Only /chunk/ requests are counted.
type fakeCDN struct {
	mu     sync.Mutex
	chunks map[string][]byte
	// script returns a non-nil reply to override the default for (host, chunkID, nth hit of that chunk).
	script    func(host, chunkID string, hit int) *cdnReply
	chunkHits map[string]int
	hostHits  map[string]int
}

func newFakeCDN() *fakeCDN {
	return &fakeCDN{
		chunks:    make(map[string][]byte),
		chunkHits: make(map[string]int),
		hostHits:  make(map[string]int),
	}
}

// add registers payload and returns a ChunkInfo (40-hex SHA-1 ID) located at offset.
func (c *fakeCDN) add(payload []byte, offset uint64) manifest.ChunkInfo {
	sum := sha1.Sum(payload)
	id := hex.EncodeToString(sum[:])
	c.chunks[id] = payload
	return manifest.ChunkInfo{ChunkID: id, Offset: offset, UncompressedSize: uint32(len(payload))}
}

func (c *fakeCDN) hits(chunkID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.chunkHits[chunkID]
}

func (c *fakeCDN) hostCount(host string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hostHits[host]
}

func (c *fakeCDN) client() *http.Client {
	return &http.Client{Transport: testutil.RoundTripFunc(c.serve)}
}

func (c *fakeCDN) serve(req *http.Request) (*http.Response, error) {
	reply := cdnReply{status: http.StatusNotFound, body: []byte("not found")}

	if idx := strings.Index(req.URL.Path, "/chunk/"); idx >= 0 {
		chunkID := strings.ToLower(req.URL.Path[idx+len("/chunk/"):])
		c.mu.Lock()
		c.chunkHits[chunkID]++
		c.hostHits[req.URL.Host]++
		hit := c.chunkHits[chunkID]
		payload, known := c.chunks[chunkID]
		script := c.script
		c.mu.Unlock()

		if known {
			reply = cdnReply{status: http.StatusOK, body: payload}
		}
		if script != nil {
			if r := script(req.URL.Host, chunkID, hit); r != nil {
				reply = *r
			}
		}
	}
	return &http.Response{
		StatusCode: reply.status,
		Body:       io.NopCloser(bytes.NewReader(reply.body)),
		Header:     make(http.Header),
	}, nil
}

func TestDownloadChunk_FailsOverFromDeadHostToHealthyHost(t *testing.T) {
	cdn := newFakeCDN()
	chunk := cdn.add([]byte("payload-served-by-the-healthy-edge"), 0)
	cdn.script = func(host, _ string, _ int) *cdnReply {
		if host == "dead.example" {
			return &cdnReply{status: http.StatusBadGateway, body: []byte("bad gateway")}
		}
		return nil
	}

	w := &testBufferWriter{}
	hosts := []string{"dead.example", "good.example"}
	if err := downloadChunk(context.Background(), cdn.client(), hosts, 1, "", chunk, w); err != nil {
		t.Fatalf("downloadChunk: %v", err)
	}
	if string(w.buf) != "payload-served-by-the-healthy-edge" {
		t.Errorf("payload = %q", w.buf)
	}
	if cdn.hostCount("good.example") == 0 {
		t.Error("healthy host was never used")
	}
}

func TestDownloadChunk_CorruptReplicaFallsBackToGoodHost(t *testing.T) {
	cdn := newFakeCDN()
	chunk := cdn.add([]byte("pristine-chunk-contents"), 0)
	cdn.script = func(host, _ string, _ int) *cdnReply {
		if host == "corrupt.example" {
			return &cdnReply{status: http.StatusOK, body: []byte("tampered-chunk-contents")}
		}
		return nil
	}

	w := &testBufferWriter{}
	hosts := []string{"corrupt.example", "good.example"}
	if err := downloadChunk(context.Background(), cdn.client(), hosts, 1, "", chunk, w); err != nil {
		t.Fatalf("downloadChunk: %v", err)
	}
	if string(w.buf) != "pristine-chunk-contents" {
		t.Errorf("checksum-failing replica leaked into output: %q", w.buf)
	}
}

func TestDownloadChunk_AllReplicasCorruptFails(t *testing.T) {
	cdn := newFakeCDN()
	chunk := cdn.add([]byte("pristine"), 0)
	cdn.script = func(string, string, int) *cdnReply {
		return &cdnReply{status: http.StatusOK, body: []byte("garbage!")}
	}

	w := &testBufferWriter{}
	err := downloadChunk(context.Background(), cdn.client(), []string{"a.example", "b.example"}, 1, "", chunk, w)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v, want checksum mismatch", err)
	}
	if len(w.buf) != 0 {
		t.Errorf("corrupt data was written: %q", w.buf)
	}
}

// singleFileManifest builds a manifest holding one file made of the given chunk payloads.
func singleFileManifest(cdn *fakeCDN, name, wrongSHA1 string, payloads ...[]byte) *manifest.Manifest {
	var all []byte
	var chunks []manifest.ChunkInfo
	for _, p := range payloads {
		chunks = append(chunks, cdn.add(p, uint64(len(all))))
		all = append(all, p...)
	}
	sum := sha1.Sum(all)
	fileSHA := hex.EncodeToString(sum[:])
	if wrongSHA1 != "" {
		fileSHA = wrongSHA1
	}
	return &manifest.Manifest{
		AppID:   9000,
		DepotID: 9001,
		Files:   []manifest.ManifestFileEntry{{Path: name, Size: uint64(len(all)), SHA1: fileSHA, Chunks: chunks}},
	}
}

func TestDownload_UnverifiableFileFailsAfterFullRedownloadFallback(t *testing.T) {
	cdn := newFakeCDN()
	m := singleFileManifest(cdn, "game.bin", strings.Repeat("0", 40), []byte("first-chunk-data"), []byte("second-chunk-data"))
	target := t.TempDir()

	_, err := DownloadOrUpdateGame(context.Background(), m, DownloadOptions{
		TargetDir: target, AppID: 9000, WorkerLimit: 1, HTTPClient: cdn.client(),
	})
	if err == nil || !strings.Contains(err.Error(), "full redownload hash mismatch") {
		t.Fatalf("err = %v, want full redownload hash mismatch", err)
	}
	for _, c := range m.Files[0].Chunks {
		if got := cdn.hits(c.ChunkID); got != 2 {
			t.Errorf("chunk %s fetched %d times, want 2 (initial pass + fallback)", c.ChunkID[:8], got)
		}
	}
}

func TestDownload_FallbackChunkFailureIsReported(t *testing.T) {
	cdn := newFakeCDN()
	m := singleFileManifest(cdn, "game.bin", strings.Repeat("f", 40), []byte("only-chunk-payload"))
	chunkID := m.Files[0].Chunks[0].ChunkID
	// The first pass succeeds (hash mismatch triggers the fallback), then the CDN dies.
	cdn.script = func(_ string, id string, hit int) *cdnReply {
		if id == chunkID && hit > 1 {
			return &cdnReply{status: http.StatusServiceUnavailable, body: []byte("down")}
		}
		return nil
	}

	_, err := DownloadOrUpdateGame(context.Background(), m, DownloadOptions{
		TargetDir: t.TempDir(), AppID: 9000, WorkerLimit: 1, HTTPClient: cdn.client(),
	})
	if err == nil || !strings.Contains(err.Error(), "download chunk during fallback") {
		t.Fatalf("err = %v, want fallback chunk failure", err)
	}
}

func TestDownload_CorruptExistingFileRecoversViaRepair(t *testing.T) {
	cdn := newFakeCDN()
	m := singleFileManifest(cdn, "game.bin", "", []byte("AAAAAAAAAAAAAAAA"), []byte("BBBBBBBBBBBBBBBB"))
	target := t.TempDir()
	path := filepath.Join(target, "game.bin")
	if err := os.WriteFile(path, []byte("AAAAAAAAAAAAAAAAXXXXXXXXXXXXXXXX"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := DownloadOrUpdateGame(context.Background(), m, DownloadOptions{
		TargetDir: target, AppID: 9000, WorkerLimit: 1, HTTPClient: cdn.client(),
	}); err != nil {
		t.Fatalf("DownloadOrUpdateGame: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "AAAAAAAAAAAAAAAABBBBBBBBBBBBBBBB" {
		t.Errorf("file = %q", got)
	}
	if hits := cdn.hits(m.Files[0].Chunks[0].ChunkID); hits != 0 {
		t.Errorf("intact chunk was refetched %d times", hits)
	}
}
