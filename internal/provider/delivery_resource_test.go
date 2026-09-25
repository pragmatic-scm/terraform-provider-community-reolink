// SPDX-License-Identifier: MPL-2.0

package provider

import "testing"

func TestScheduleWithEnabledLegacy(t *testing.T) {
	const disabled = `{"enable":0,"table":"1111"}`
	const enabled = `{"enable":1,"table":"1111"}`

	got, changed, err := scheduleWithEnabled(disabled, true)
	if err != nil {
		t.Fatalf("scheduleWithEnabled returned an error: %v", err)
	}
	if !changed {
		t.Fatal("scheduleWithEnabled did not report a changed legacy schedule")
	}
	if got != enabled {
		t.Fatalf("scheduleWithEnabled = %q, want %q", got, enabled)
	}

	got, changed, err = scheduleWithEnabled(enabled, true)
	if err != nil {
		t.Fatalf("scheduleWithEnabled returned an error: %v", err)
	}
	if changed || got != enabled {
		t.Fatalf("already enabled schedule = %q, changed %t; want unchanged %q", got, changed, enabled)
	}
}

func TestScheduleWithEnabledLeavesV20ScheduleUnchanged(t *testing.T) {
	const schedule = `{"channel":0,"table":{"AI_PEOPLE":"1111"}}`

	got, changed, err := scheduleWithEnabled(schedule, true)
	if err != nil {
		t.Fatalf("scheduleWithEnabled returned an error: %v", err)
	}
	if changed || got != schedule {
		t.Fatalf("V20 schedule = %q, changed %t; want unchanged %q", got, changed, schedule)
	}
}

func TestScheduleWithEnabledRejectsInvalidJSON(t *testing.T) {
	if _, _, err := scheduleWithEnabled(`{"enable":`, true); err == nil {
		t.Fatal("scheduleWithEnabled accepted invalid JSON")
	}
}
