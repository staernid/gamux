package downloader

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/util"
)

// NewDownloaderHTTPClient returns an http.Client configured with dedicated transport pooling
// optimized for high-concurrency depot chunk downloads.
func NewDownloaderHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: transport,
	}
}

// DefaultHTTPClient is the default http.Client used for downloader operations.
var DefaultHTTPClient = NewDownloaderHTTPClient()

type cdnServerEntry struct {
	Host          string `json:"host"`
	VHost         string `json:"vhost"`
	Load          int    `json:"load"`
	WeightedLoad  int    `json:"weighted_load"`
	PriorityClass int    `json:"priority_class"`
	HTTPSSupport  string `json:"https_support"`
}

type cdnServerResponse struct {
	Response struct {
		Servers []cdnServerEntry `json:"servers"`
	} `json:"response"`
}

// rankCDNEdges measures HTTP round-trip latency to candidate edge hosts concurrently and returns hosts ordered by speed.
func rankCDNEdges(ctx context.Context, client *http.Client, hosts []string) []string {
	if client == nil {
		client = DefaultHTTPClient
	}
	if len(hosts) <= 1 {
		return hosts
	}

	type hostLatency struct {
		host string
		lat  time.Duration
	}

	latCh := make(chan hostLatency, len(hosts))
	timeoutCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	for _, h := range hosts {
		host := h
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			req, err := http.NewRequestWithContext(timeoutCtx, http.MethodHead, "https://"+host, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", "gamux/1.0")
			resp, err := client.Do(req)
			if err == nil {
				_ = resp.Body.Close()
				latCh <- hostLatency{host: host, lat: time.Since(start)}
			}
		}()
	}

	wg.Wait()
	close(latCh)

	var responsive []hostLatency
	for hl := range latCh {
		responsive = append(responsive, hl)
	}

	if len(responsive) == 0 {
		return hosts
	}

	sort.Slice(responsive, func(i, j int) bool {
		return responsive[i].lat < responsive[j].lat
	})

	ranked := make([]string, 0, len(hosts))
	seen := make(map[string]bool)
	for _, r := range responsive {
		ranked = append(ranked, r.host)
		seen[r.host] = true
	}
	for _, h := range hosts {
		if !seen[h] {
			ranked = append(ranked, h)
		}
	}
	return ranked
}

// fetchCDNServers discovers active SteamPipe CDN edge servers via Valve's directory Web API.
func fetchCDNServers(ctx context.Context, client *http.Client) ([]string, error) {
	if client == nil {
		client = DefaultHTTPClient
	}
	url := "https://api.steampowered.com/IContentServerDirectoryService/GetServersForSteamPipe/v1/?cell_id=0"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create cdn query request: %w", err)
	}
	req.Header.Set("User-Agent", "gamux/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cdn query request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cdn directory service HTTP status %d", resp.StatusCode)
	}

	var payload cdnServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode cdn response json: %w", err)
	}

	var validServers []cdnServerEntry
	for _, s := range payload.Response.Servers {
		h := strings.TrimSpace(s.Host)
		if h == "" {
			h = strings.TrimSpace(s.VHost)
		}
		if h == "" {
			continue
		}
		if strings.EqualFold(s.HTTPSSupport, "unsupported") || strings.EqualFold(s.HTTPSSupport, "none") {
			continue
		}
		s.Host = h
		validServers = append(validServers, s)
	}

	if len(validServers) == 0 {
		return nil, fmt.Errorf("no cdn hosts returned by SteamPipe service")
	}

	// Rank servers by Valve's official metrics:
	// 1. Primary sort: PriorityClass DESC (Valve's preferred tier)
	// 2. Secondary sort: WeightedLoad ASC (least congested node)
	sort.SliceStable(validServers, func(i, j int) bool {
		if validServers[i].PriorityClass != validServers[j].PriorityClass {
			return validServers[i].PriorityClass > validServers[j].PriorityClass
		}
		return validServers[i].WeightedLoad < validServers[j].WeightedLoad
	})

	limit := 5
	if len(validServers) < limit {
		limit = len(validServers)
	}

	topHosts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		topHosts = append(topHosts, validServers[i].Host)
	}

	return topHosts, nil
}

func downloadChunk(ctx context.Context, client *http.Client, cdnHosts []string, depotID uint32, decKey string, chunk manifest.ChunkInfo, dst io.WriterAt) error {
	if client == nil {
		client = DefaultHTTPClient
	}
	if len(cdnHosts) == 0 {
		cdnHosts = []string{"cache10-fra1.steamcontent.com", "cache4-fra1.steamcontent.com", "cache1-fra1.steamcontent.com"}
	}

	chunkID := strings.ToLower(chunk.ChunkID)
	startIdx := 0
	if len(cdnHosts) > 1 && len(chunkID) >= 4 {
		if n, err := strconv.ParseUint(chunkID[:4], 16, 32); err == nil {
			startIdx = int(n) % len(cdnHosts)
		}
	}

	const maxAttempts = 3
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if attempt > 0 {
			jitter := time.Duration(10+int(chunkID[0])%50) * time.Millisecond
			backoff := time.Duration(150*(1<<attempt))*time.Millisecond + jitter
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		for i := 0; i < len(cdnHosts); i++ {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			host := cdnHosts[(startIdx+i+attempt)%len(cdnHosts)]
			chunkURL := fmt.Sprintf("https://%s/depot/%d/chunk/%s", host, depotID, chunkID)

			reqCtx, reqCancel := context.WithTimeout(ctx, 45*time.Second)
			req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, chunkURL, nil)
			if err != nil {
				reqCancel()
				lastErr = err
				continue
			}
			req.Header.Set("User-Agent", "Valve/Steam HTTP Client 1.0")

			resp, err := client.Do(req)
			if err != nil {
				reqCancel()
				if ctx.Err() != nil {
					return ctx.Err()
				}
				lastErr = fmt.Errorf("fetch from %s: %w", host, err)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				_, _ = io.CopyN(io.Discard, resp.Body, 4096)
				_ = resp.Body.Close()
				reqCancel()
				lastErr = fmt.Errorf("HTTP status %d from %s", resp.StatusCode, host)
				continue
			}

			rawChunkData, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			reqCancel()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				lastErr = fmt.Errorf("read chunk body from %s: %w", host, err)
				continue
			}

			// 1. Decrypt chunk payload if encrypted with depot key
			payloadData, err := manifest.DecryptChunkPayload(rawChunkData, decKey)
			if err != nil {
				lastErr = fmt.Errorf("decrypt chunk %s: %w", chunk.ChunkID, err)
				continue
			}

			// 2. Decompress chunk payload
			decompressed, err := decompressChunkData(payloadData, chunk.UncompressedSize)
			if err != nil {
				lastErr = fmt.Errorf("decompress chunk %s: %w", chunk.ChunkID, err)
				continue
			}

			// 3. Verify chunk integrity against chunk.ChunkID SHA-1 (when 40-hex SHA1 hash is provided)
			if len(chunk.ChunkID) == 40 {
				chunkSum := sha1.Sum(decompressed)
				chunkHashHex := hex.EncodeToString(chunkSum[:])
				if !strings.EqualFold(chunkHashHex, chunk.ChunkID) {
					lastErr = fmt.Errorf("chunk %s checksum mismatch: got %s", chunk.ChunkID, chunkHashHex)
					continue
				}
			}

			// 4. Write chunk payload at expected offset
			if _, err := dst.WriteAt(decompressed, int64(chunk.Offset)); err != nil {
				return fmt.Errorf("write chunk %s at offset %d: %w", chunk.ChunkID, chunk.Offset, err)
			}

			return nil
		}

		slog.Warn("Retrying chunk download after host round failure",
			"chunkID", chunk.ChunkID,
			"depotID", depotID,
			"attempt", attempt+1,
			"maxAttempts", maxAttempts,
			"lastErr", lastErr,
		)
	}

	slog.Error("Failed to download chunk after all retry attempts",
		"chunkID", chunk.ChunkID,
		"depotID", depotID,
		"hosts", len(cdnHosts),
		"error", lastErr,
	)
	return fmt.Errorf("download chunk %s (depot %d) failed after %d attempts across %d CDN hosts: %w",
		chunk.ChunkID, depotID, maxAttempts, len(cdnHosts), lastErr)
}

func decompressChunkData(data []byte, expectedSize uint32) ([]byte, error) {
	return util.DecompressChunkSlice(data, expectedSize)
}
