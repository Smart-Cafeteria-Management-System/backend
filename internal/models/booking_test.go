package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestBookingStatusConstants(t *testing.T) {
	tests := []struct {
		status   BookingStatus
		expected string
	}{
		{BookingConfirmed, "confirmed"},
		{BookingServed, "served"},
		{BookingCancelled, "cancelled"},
		{BookingNoShow, "no-show"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected status %q, got %q", tt.expected, tt.status)
		}
	}
}

func TestMealTypeConstants(t *testing.T) {
	tests := []struct {
		meal     MealType
		expected string
	}{
		{MealBreakfast, "breakfast"},
		{MealLunch, "lunch"},
		{MealSnacks, "snacks"},
		{MealDinner, "dinner"},
	}

	for _, tt := range tests {
		if string(tt.meal) != tt.expected {
			t.Errorf("expected meal type %q, got %q", tt.expected, tt.meal)
		}
	}
}

func TestSlotStatusConstants(t *testing.T) {
	tests := []struct {
		status   SlotStatus
		expected string
	}{
		{SlotAvailable, "available"},
		{SlotFull, "full"},
		{SlotClosed, "closed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected slot status %q, got %q", tt.expected, tt.status)
		}
	}
}

func TestItemCategoryConstants(t *testing.T) {
	tests := []struct {
		cat      ItemCategory
		expected string
	}{
		{CategoryMain, "main"},
		{CategorySide, "side"},
		{CategoryBeverage, "beverage"},
		{CategoryDessert, "dessert"},
	}

	for _, tt := range tests {
		if string(tt.cat) != tt.expected {
			t.Errorf("expected category %q, got %q", tt.expected, tt.cat)
		}
	}
}

func TestBookingBeforeCreateSetsUUID(t *testing.T) {
	booking := &Booking{}
	err := booking.BeforeCreate(nil)
	if err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	if booking.ID == uuid.Nil {
		t.Error("BeforeCreate should set a non-nil UUID")
	}
}

func TestBookingBeforeCreatePreservesExistingUUID(t *testing.T) {
	existingID := uuid.New()
	booking := &Booking{ID: existingID}
	err := booking.BeforeCreate(nil)
	if err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	if booking.ID != existingID {
		t.Error("BeforeCreate should not overwrite an existing UUID")
	}
}

func TestMenuItemBeforeCreateSetsUUID(t *testing.T) {
	item := &MenuItem{}
	err := item.BeforeCreate(nil)
	if err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	if item.ID == uuid.Nil {
		t.Error("BeforeCreate should set a non-nil UUID")
	}
}

func TestMealSlotBeforeCreateSetsUUID(t *testing.T) {
	slot := &MealSlot{}
	err := slot.BeforeCreate(nil)
	if err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	if slot.ID == uuid.Nil {
		t.Error("BeforeCreate should set a non-nil UUID")
	}
}
