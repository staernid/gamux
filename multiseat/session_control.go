package multiseat

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/util"
)

func (s *Supervisor) superviseSession(gameDir string, session *runningSession) {
	s.mu.Lock()
	seats := make([]*runningSeat, 0, len(session.seats))
	for _, seat := range session.seats {
		seats = append(seats, seat)
	}
	s.mu.Unlock()

	type seatExit struct {
		seatID string
		seat   *runningSeat
		err    error
	}

	exitCh := make(chan seatExit, len(seats))
	for _, seat := range seats {
		st := seat
		go func() {
			err := st.cmd.Wait()
			exitCh <- seatExit{seatID: st.seatID, seat: st, err: err}
		}()
	}

	activeCount := len(seats)
	var lastExitErr error

	for activeCount > 0 {
		ex := <-exitCh
		activeCount--
		if ex.err != nil {
			lastExitErr = ex.err
		}

		s.mu.Lock()
		ex.seat.status.IsRunning = false
		link := ex.seat.waylandLink
		ex.seat.waylandLink = ""
		logF := ex.seat.logFile
		ex.seat.logFile = nil
		eb := s.bus
		s.mu.Unlock()

		slog.Info("Multi-seat Seat terminated", "dir", gameDir, "seat", ex.seatID, "profile", ex.seat.profileID, "err", ex.err)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if ex.seat.grabber != nil {
			ex.seat.grabber.Release()
		}
		if ex.seat.sink != nil {
			ex.seat.sink.Destroy(ctx)
		}
		if link != "" {
			_ = os.Remove(link)
		}
		if ex.seat.sunshineCmd != nil && ex.seat.sunshineCmd.Process != nil {
			_ = ex.seat.sunshineCmd.Process.Kill()
		}
		if logF != nil {
			_ = logF.Close()
		}
		cancel()

		if eb != nil {
			eb.Publish(events.NewGameInstanceExited(gameDir, session.config.Name, session.config.AppID, ex.err, ex.seat.profileID, ex.seatID))
		}

		// If a seat exited unexpectedly while other seats are still running
		if (ex.err != nil || time.Now().Unix()-session.startTime < 10) && activeCount > 0 {
			if eb != nil {
				msg := fmt.Sprintf("Player seat %s exited unexpectedly.", ex.seatID)
				if snippet := readLogTail(ex.seat.logPath, 15); snippet != "" {
					msg += "\n\nLog tail:\n" + snippet
				}
				eb.Publish(events.NewNotification(fmt.Sprintf("Multi-Seat %s Error", strings.ToUpper(ex.seatID)), msg))
			}
		}
	}

	duration := time.Now().Unix() - session.startTime

	s.mu.Lock()
	eb := s.bus
	s.mu.Unlock()

	if eb != nil {
		if lastExitErr != nil || duration < 10 {
			msg := fmt.Sprintf("Multi-Seat Session terminated after %ds.", duration)
			eb.Publish(events.NewNotification("Multi-Seat Session Finished", msg))
		}
		eb.Publish(events.NewGameExited(gameDir, session.config.Name, session.config.AppID, lastExitErr))
	}

	// Clean up remaining session state
	s.mu.Lock()
	activeSession := s.sessions[gameDir]
	s.mu.Unlock()

	if activeSession != nil {
		_ = s.StopSession(gameDir)
	}
}

// StopSession cleanly terminates all running seats, sinks, relays, and sunshine servers for a game.
func (s *Supervisor) StopSession(gameDir string) error {
	gameDir = util.CanonicalPath(gameDir)

	s.mu.Lock()
	session, exists := s.sessions[gameDir]
	if !exists {
		s.mu.Unlock()
		return nil
	}
	delete(s.sessions, gameDir)
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.mu.Lock()
	for _, seat := range session.seats {
		seat.status.IsRunning = false
	}
	s.mu.Unlock()

	for _, seat := range session.seats {
		if seat.grabber != nil {
			seat.grabber.Release()
		}
		if seat.pgid > 1 {
			_ = syscall.Kill(-seat.pgid, syscall.SIGTERM)
		} else if seat.cmd != nil && seat.cmd.Process != nil {
			_ = seat.cmd.Process.Kill()
		}
		if seat.sunshineCmd != nil && seat.sunshineCmd.Process != nil {
			_ = seat.sunshineCmd.Process.Kill()
		}
		if seat.sink != nil {
			seat.sink.Destroy(ctx)
		}
		s.mu.Lock()
		logF := seat.logFile
		seat.logFile = nil
		link := seat.waylandLink
		seat.waylandLink = ""
		s.mu.Unlock()
		if logF != nil {
			_ = logF.Close()
		}
		if link != "" {
			_ = os.Remove(link)
		}
	}

	// Cleanup KWin splitscreen coordinator if active
	if session.kwinCoord != nil {
		_ = session.kwinCoord.Stop()
	}

	// Restore display scales if no other sessions are active
	s.mu.Lock()
	sessionsLeft := len(s.sessions)
	s.mu.Unlock()
	if sessionsLeft == 0 && s.scaleGuard != nil {
		_ = s.scaleGuard.Restore(ctx)
	}

	slog.Info("Multi-seat session cleanly terminated", "dir", gameDir)
	return nil
}

// StopSeat terminates only a single player seat in the session without killing others.
func (s *Supervisor) StopSeat(gameDir string, seatID string) error {
	gameDir = util.CanonicalPath(gameDir)

	s.mu.Lock()
	session, exists := s.sessions[gameDir]
	s.mu.Unlock()

	if !exists || session == nil {
		return fmt.Errorf("no active multi-seat session for %s", gameDir)
	}

	s.mu.Lock()
	seat, found := session.seats[seatID]
	s.mu.Unlock()

	if !found || seat == nil {
		return fmt.Errorf("seat %s not found in session for %s", seatID, gameDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if seat.grabber != nil {
		seat.grabber.Release()
	}
	if seat.pgid > 1 {
		_ = syscall.Kill(-seat.pgid, syscall.SIGTERM)
	} else if seat.cmd != nil && seat.cmd.Process != nil {
		_ = seat.cmd.Process.Kill()
	}
	if seat.sunshineCmd != nil && seat.sunshineCmd.Process != nil {
		_ = seat.sunshineCmd.Process.Kill()
	}
	if seat.sink != nil {
		seat.sink.Destroy(ctx)
	}
	s.mu.Lock()
	seat.status.IsRunning = false
	logF := seat.logFile
	seat.logFile = nil
	link := seat.waylandLink
	seat.waylandLink = ""
	s.mu.Unlock()

	if logF != nil {
		_ = logF.Close()
	}
	if link != "" {
		_ = os.Remove(link)
	}

	return nil
}

// GetStatus returns the live multi-seat session status for a game.
func (s *Supervisor) GetStatus(gameDir string) *MultiseatSessionStatus {
	gameDir = util.CanonicalPath(gameDir)

	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[gameDir]
	if !exists || session == nil {
		return &MultiseatSessionStatus{
			GameDir:   gameDir,
			IsRunning: false,
		}
	}

	seatsStatus := make(map[string]SeatStatus, len(session.seats))
	isRunning := false
	for id, st := range session.seats {
		seatsStatus[id] = st.status
		if st.status.IsRunning {
			isRunning = true
		}
	}

	status := &MultiseatSessionStatus{
		GameDir:   gameDir,
		IsRunning: isRunning,
		Seats:     seatsStatus,
		StartTime: session.startTime,
	}

	// Populate backward-compatible Seat1/Seat2
	if s1, ok := seatsStatus["p1"]; ok {
		status.Seat1 = s1
	}
	if s2, ok := seatsStatus["p2"]; ok {
		status.Seat2 = s2
	}

	return status
}

// IsSessionRunning checks if a multi-seat session is active for the game directory.
func (s *Supervisor) IsSessionRunning(gameDir string) bool {
	st := s.GetStatus(gameDir)
	return st != nil && st.IsRunning
}

// RunningSessions returns canonical paths of all active multi-seat sessions.
func (s *Supervisor) RunningSessions() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := make([]string, 0, len(s.sessions))
	for dir, sess := range s.sessions {
		if sess != nil {
			for _, seat := range sess.seats {
				if seat.status.IsRunning {
					list = append(list, util.CanonicalPath(dir))
					break
				}
			}
		}
	}
	return list
}

// GetRunningInstances returns all active seat instances across all multiseat sessions.
func (s *Supervisor) GetRunningInstances() []domain.RunningInstance {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []domain.RunningInstance
	for gameDir, sess := range s.sessions {
		if sess == nil {
			continue
		}
		for seatID, seat := range sess.seats {
			if seat.status.IsRunning && seat.cmd != nil && seat.cmd.Process != nil {
				inst := domain.RunningInstance{
					InstanceID:  domain.MakeInstanceID(gameDir, seat.profileID),
					GameDir:     gameDir,
					GameName:    sess.config.Name,
					AppID:       sess.config.AppID,
					ProfileID:   seat.profileID,
					SeatID:      seatID,
					PID:         seat.cmd.Process.Pid,
					IsMultiseat: true,
					Mode:        string(seat.status.Mode),
					StartTime:   sess.startTime,
				}
				result = append(result, inst)
			}
		}
	}
	return result
}
