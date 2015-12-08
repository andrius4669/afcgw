package main

import "testing"

func TestMakeTrip(t *testing.T) {
	type tripset struct {
		src  string
		name string
		trip string
	}
	var tests = [...]tripset{
		{ src: "",                  name: "",     trip: "" },
		{ src: "test",              name: "test", trip: "" },
		{ src: "#:^)",              name: "",     trip: "!qbhz/q8HqQ" },
		{ src: "asda#a6516a51aaaa", name: "asda", trip: "!Om/F889ywA" },
		{ src: "bbb#猫に哲学",       name: "bbb",  trip: "!tcVgirItgw" },
	}
	for i := range tests {
		name, trip := MakeTrip(tests[i].src)
		if name != tests[i].name {
			t.Errorf("name: expected: %s; got: %s\n", tests[i].name, name)
		}
		if trip != tests[i].trip {
			t.Errorf("trip: expected: %s; got: %s\n", tests[i].trip, trip)
		}
	}
}