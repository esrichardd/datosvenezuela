package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/esrichardd/datosvenezuela/api/internal/banks"
	"github.com/esrichardd/datosvenezuela/api/internal/geo"
	"github.com/esrichardd/datosvenezuela/api/internal/holidays"
	"github.com/esrichardd/datosvenezuela/api/internal/phones"
	"github.com/esrichardd/datosvenezuela/api/internal/platform/store"
)

// stores holds the data source of every domain the API serves.
type stores struct {
	geo      *geo.CSVStore
	banks    *banks.CSVStore
	phones   *phones.CSVStore
	holidays *holidays.CSVStore
}

// loadStores loads and validates every dataset in dataDir. It stops at the
// first dataset that fails, naming it in the error.
func loadStores(ctx context.Context, dataDir string) (stores, error) {
	var s stores
	var err error

	if s.geo, err = geo.LoadCSV(filepath.Join(dataDir, "geography")); err != nil {
		return stores{}, fmt.Errorf("load geography: %w", err)
	}
	if s.banks, err = banks.LoadCSV(filepath.Join(dataDir, "banks")); err != nil {
		return stores{}, fmt.Errorf("load banks: %w", err)
	}
	// Phones refers to the states of geography, so it is loaded after it.
	if s.phones, err = phones.LoadCSV(ctx, filepath.Join(dataDir, "phones"), s.geo); err != nil {
		return stores{}, fmt.Errorf("load phones: %w", err)
	}
	if s.holidays, err = holidays.LoadCSV(filepath.Join(dataDir, "holidays")); err != nil {
		return stores{}, fmt.Errorf("load holidays: %w", err)
	}

	return s, nil
}

// logDataLoaded logs how many records of each dataset were loaded, to confirm
// at startup that the expected data is being served. An empty Page selects no
// items, so each call only returns the total.
func logDataLoaded(ctx context.Context, logger *slog.Logger, s stores) {
	_, states, _ := s.geo.States(ctx, store.Page{})
	_, municipalities, _ := s.geo.Municipalities(ctx, store.Page{})
	_, parishes, _ := s.geo.Parishes(ctx, store.Page{})
	_, bankCount, _ := s.banks.Banks(ctx, store.Page{})
	_, phonePrefixes, _ := s.phones.Prefixes(ctx, "", store.Page{})
	bankHolidayYears, _ := s.holidays.BankYears(ctx)
	logger.Info("data loaded",
		"states", states,
		"municipalities", municipalities,
		"parishes", parishes,
		"banks", bankCount,
		"phone_prefixes", phonePrefixes,
		"bank_holiday_years", bankHolidayYears,
	)
}
