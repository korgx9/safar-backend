package tests

import (
	"net/http"
	"testing"

	"github.com/korgx9/safar-backend/internal/models"
)

func TestVehicleAuthenticatedUserCanCreateVehicle(t *testing.T) {
	db, router := setupTestEnv(t)
	user, token := createTestUserAndToken(t, db, "+992900002001", models.UserRolePassenger)

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/vehicles",
		map[string]interface{}{"seats_total": 4},
		token,
	)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", w.Code, w.Body.String())
	}

	vehicle := decodeJSON[models.Vehicle](t, w)
	if vehicle.UserID != user.ID {
		t.Fatalf("expected user_id=%d, got %d", user.ID, vehicle.UserID)
	}
	if vehicle.SeatsTotal != 4 {
		t.Fatalf("expected seats_total=4, got %d", vehicle.SeatsTotal)
	}
}

func TestVehicleAuthenticatedUserCanListOwnVehicles(t *testing.T) {
	db, router := setupTestEnv(t)
	user1, token1 := createTestUserAndToken(t, db, "+992900002002", models.UserRolePassenger)
	user2, _ := createTestUserAndToken(t, db, "+992900002003", models.UserRolePassenger)

	createVehicle(t, db, user1.ID, 4)
	createVehicle(t, db, user1.ID, 6)
	createVehicle(t, db, user2.ID, 2)

	w := performRequest(t, router, http.MethodGet, "/vehicles", nil, token1)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	vehicles := decodeJSON[[]models.Vehicle](t, w)
	if len(vehicles) != 2 {
		t.Fatalf("expected 2 vehicles, got %d, body=%s", len(vehicles), w.Body.String())
	}

	for _, vehicle := range vehicles {
		if vehicle.UserID != user1.ID {
			t.Fatalf("expected user_id=%d, got %d", user1.ID, vehicle.UserID)
		}
	}
}

func TestVehicleInvalidSeatsTotalReturns400(t *testing.T) {
	db, router := setupTestEnv(t)
	_, token := createTestUserAndToken(t, db, "+992900002004", models.UserRolePassenger)

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/vehicles",
		map[string]interface{}{"seats_total": 0},
		token,
	)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}
