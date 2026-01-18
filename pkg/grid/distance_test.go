package grid

import "testing"

func TestDistances(t *testing.T) {
	a := TilePos{0, 0}
	b := TilePos{3, 4}

	if a.DistanceSquared(b) != 25 {
		t.Fatalf("DistanceSquared failed")
	}

	if a.ManhattanDistance(b) != 7 {
		t.Fatalf("ManhattanDistance failed")
	}

	if a.ChebyshevDistance(b) != 4 {
		t.Fatalf("ChebyshevDistance failed")
	}

	if a.OctileDistance(b) != 10*1+14*3 {
		t.Fatalf("OctileDistance unexpected")
	}
}

func TestDistanceSymmetry(t *testing.T) {
	a := TilePos{10, -5}
	b := TilePos{-3, 7}

	if a.DistanceSquared(b) != b.DistanceSquared(a) {
		t.Fatal("DistanceSquared not symmetric")
	}
}
