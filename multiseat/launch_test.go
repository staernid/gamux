package multiseat

import "testing"

func TestNormalizeLayout(t *testing.T) {
	tests := []struct {
		name   string
		layout DisplayLayout
		seats  int
		want   DisplayLayout
	}{
		{"two seats quad becomes side by side", LayoutQuadGrid, 2, LayoutSideBySide},
		{"two seats three-player becomes side by side", LayoutThreePlayer, 2, LayoutSideBySide},
		{"three seats quad becomes three-player", LayoutQuadGrid, 3, LayoutThreePlayer},
		{"four seats quad unchanged", LayoutQuadGrid, 4, LayoutQuadGrid},
		{"two seats top-bottom unchanged", LayoutTopBottom, 2, LayoutTopBottom},
		{"single seat unchanged", LayoutQuadGrid, 1, LayoutQuadGrid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeLayout(tt.layout, tt.seats); got != tt.want {
				t.Errorf("normalizeLayout(%q, %d) = %q, want %q", tt.layout, tt.seats, got, tt.want)
			}
		})
	}
}

func TestClampDisplayIndices(t *testing.T) {
	seats := []SeatConfig{{DisplayIndex: 5}, {DisplayIndex: -1}, {DisplayIndex: 1}}
	clampDisplayIndices(seats, 2, LayoutWindowed)
	want := []int{0, 0, 1}
	for i, w := range want {
		if seats[i].DisplayIndex != w {
			t.Errorf("seat %d display = %d, want %d", i, seats[i].DisplayIndex, w)
		}
	}

	// A split layout on a single display pins everything to display 0.
	seats = []SeatConfig{{DisplayIndex: 0}, {DisplayIndex: 0}}
	clampDisplayIndices(seats, 1, LayoutSideBySide)
	for i := range seats {
		if seats[i].DisplayIndex != 0 {
			t.Errorf("seat %d display = %d, want 0", i, seats[i].DisplayIndex)
		}
	}
}

func TestSeatSlot(t *testing.T) {
	two, neg, big := 2, -1, 9
	tests := []struct {
		name string
		slot *int
		want int
	}{
		{"nil falls back to index", nil, 1},
		{"explicit slot wins", &two, 2},
		{"negative falls back", &neg, 1},
		{"out of range falls back", &big, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := seatSlot(SeatConfig{ScreenSlot: tt.slot}, 1, 4); got != tt.want {
				t.Errorf("seatSlot = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSunshinePortFor(t *testing.T) {
	cfg := MultiseatSessionConfig{SunshinePortP1: 1111, SunshinePortP2: 2222}
	tests := []struct {
		name string
		seat SeatConfig
		idx  int
		want int
	}{
		{"seat override", SeatConfig{SunshinePort: 5000}, 0, 5000},
		{"p1 session port", SeatConfig{}, 0, 1111},
		{"p2 session port", SeatConfig{}, 1, 2222},
		{"default stride", SeatConfig{}, 2, 47989 + 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sunshinePortFor(tt.seat, cfg, tt.idx); got != tt.want {
				t.Errorf("sunshinePortFor = %d, want %d", got, tt.want)
			}
		})
	}
}
