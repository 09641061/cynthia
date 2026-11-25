//go:build integration

package calendar_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/calendar/domain"
	"go-service/internal/calendar/infrastructure/persistence/postgres"
	"go-service/internal/catalog"
	"go-service/internal/crm"
	"go-service/internal/shared/circuitbreaker"
	sharedpostgres "go-service/internal/shared/postgres"
)

func TestExclusionConstraintAndConcurrentOverlap(t *testing.T) {
	db, refs := newTestDB(t)
	store := postgres.NewStore(db)
	ctx := t.Context()
	start := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)

	first, err := domain.NewAppointment(domain.CreateInput{
		ID: uuid.NewString(), PersonID: refs.personA, CustomerID: refs.customerID, ServiceID: refs.serviceID,
		StartsAt: start, EndsAt: start.Add(time.Hour), Status: domain.StatusConfirmed,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateAppointment(ctx, first); err != nil {
		t.Fatalf("create first appointment: %v", err)
	}

	// Adjacent slot [11:00, 12:00) for same person must succeed.
	adjacent, _ := domain.NewAppointment(domain.CreateInput{
		ID: uuid.NewString(), PersonID: refs.personA, CustomerID: refs.customerID, ServiceID: refs.serviceID,
		StartsAt: start.Add(time.Hour), EndsAt: start.Add(2 * time.Hour), Status: domain.StatusPending,
	})
	if _, err := store.CreateAppointment(ctx, adjacent); err != nil {
		t.Fatalf("adjacent slot must succeed: %v", err)
	}

	// Different person at same time [10:00, 11:00) must succeed.
	otherPerson, _ := domain.NewAppointment(domain.CreateInput{
		ID: uuid.NewString(), PersonID: refs.personB, CustomerID: refs.customerID, ServiceID: refs.serviceID,
		StartsAt: start, EndsAt: start.Add(time.Hour), Status: domain.StatusConfirmed,
	})
	if _, err := store.CreateAppointment(ctx, otherPerson); err != nil {
		t.Fatalf("different person slot must succeed: %v", err)
	}

	// Concurrent overlapping creates for personB at [14:00, 15:00): exactly 1 succeeds, 1 gets ErrTimeConflict.
	raceStart := time.Date(2026, 11, 25, 14, 0, 0, 0, time.UTC)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	ready := make(chan struct{})
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ready
			appt, _ := domain.NewAppointment(domain.CreateInput{
				ID: uuid.NewString(), PersonID: refs.personB, CustomerID: refs.customerID, ServiceID: refs.serviceID,
				StartsAt: raceStart, EndsAt: raceStart.Add(time.Hour), Status: domain.StatusPending,
			})
			_, err := store.CreateAppointment(ctx, appt)
			errs <- err
		}()
	}
	close(ready)
	wg.Wait()
	close(errs)

	var successes, conflicts int
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrTimeConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent overlap: successes=%d conflicts=%d, want 1 and 1", successes, conflicts)
	}
}

func TestCrossContextReadersAndCancelledSlotReuse(t *testing.T) {
	db, refs := newTestDB(t)
	ctx := t.Context()
	breaker, err := circuitbreaker.New(5, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client := sharedpostgres.NewClient(db, breaker)

	custReader := crm.NewCustomerReader(client)
	cust, err := custReader.GetCustomer(ctx, refs.customerID)
	if err != nil || cust.Name != "Ana" {
		t.Fatalf("GetCustomer: %v, %+v", err, cust)
	}
	custMap, err := custReader.GetCustomersByIDs(ctx, []string{refs.customerID})
	if err != nil || custMap[refs.customerID].Phone != "+573001112233" {
		t.Fatalf("GetCustomersByIDs: %v, %+v", err, custMap)
	}

	svcReader := catalog.NewServiceReader(client)
	svc, err := svcReader.GetService(ctx, refs.serviceID)
	if err != nil || svc.DurationMinutes != 30 {
		t.Fatalf("GetService: %v, %+v", err, svc)
	}
	svcMap, err := svcReader.GetServicesByIDs(ctx, []string{refs.serviceID})
	if err != nil || svcMap[refs.serviceID].Name != "Cut" {
		t.Fatalf("GetServicesByIDs: %v, %+v", err, svcMap)
	}

	// Cancelling an appointment frees the slot for a new booking.
	store := postgres.NewStore(db)
	start := time.Date(2026, 11, 25, 16, 0, 0, 0, time.UTC)
	appt, _ := domain.NewAppointment(domain.CreateInput{
		ID: uuid.NewString(), PersonID: refs.personA, CustomerID: refs.customerID, ServiceID: refs.serviceID,
		StartsAt: start, EndsAt: start.Add(time.Hour), Status: domain.StatusConfirmed,
	})
	saved, err := store.CreateAppointment(ctx, appt)
	if err != nil {
		t.Fatal(err)
	}
	cancelled := domain.StatusCancelled
	patched, _ := saved.ApplyPatch(domain.Patch{Status: &cancelled})
	if _, err := store.UpdateAppointment(ctx, 1, patched); err != nil {
		t.Fatal(err)
	}
	rebooked, _ := domain.NewAppointment(domain.CreateInput{
		ID: uuid.NewString(), PersonID: refs.personA, CustomerID: refs.customerID, ServiceID: refs.serviceID,
		StartsAt: start, EndsAt: start.Add(time.Hour), Status: domain.StatusPending,
	})
	if _, err := store.CreateAppointment(ctx, rebooked); err != nil {
		t.Fatalf("rebooking cancelled slot failed: %v", err)
	}
}
