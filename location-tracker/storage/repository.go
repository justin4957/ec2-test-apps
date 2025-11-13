/*
# Module: storage/repository.go
Repository interfaces for data persistence layer.

## Linked Modules
- [types/error_log](../types/error_log.go) - Error log data structures
- [types/location](../types/location.go) - Location data structures
- [types/commercial](../types/commercial.go) - Commercial real estate data structures
- [types/tip](../types/tip.go) - Anonymous tip data structures

## Tags
storage, repository, interface, persistence

## Exports
ErrorLogRepository, LocationRepository, CommercialRepository, TipRepository

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "storage/repository.go" ;
    code:description "Repository interfaces for data persistence layer" ;
    code:linksTo [
        code:name "types/error_log" ;
        code:path "../types/error_log.go" ;
        code:relationship "Error log data structures"
    ], [
        code:name "types/location" ;
        code:path "../types/location.go" ;
        code:relationship "Location data structures"
    ], [
        code:name "types/commercial" ;
        code:path "../types/commercial.go" ;
        code:relationship "Commercial real estate data structures"
    ], [
        code:name "types/tip" ;
        code:path "../types/tip.go" ;
        code:relationship "Anonymous tip data structures"
    ] ;
    code:exports :ErrorLogRepository, :LocationRepository, :CommercialRepository, :TipRepository ;
    code:tags "storage", "repository", "interface", "persistence" .
<!-- End LinkedDoc RDF -->
*/
package storage

import (
	"location-tracker/types"
)

// ErrorLogRepository handles error log persistence
type ErrorLogRepository interface {
	Save(errorLog types.ErrorLog) error
	GetByID(id string) (*types.ErrorLog, error)
	GetRecent(limit int) ([]types.ErrorLog, error)
	GetAll() ([]types.ErrorLog, error)
}

// LocationRepository handles location persistence
type LocationRepository interface {
	Save(location types.Location) error
	GetByDeviceID(deviceID string) (*types.Location, error)
	GetAll() (map[string]types.Location, error)
}

// CommercialRepository handles commercial real estate persistence
type CommercialRepository interface {
	Save(commercial types.CommercialRealEstate) error
	GetByLocation(lat, lng float64, radiusMiles float64) (*types.CommercialRealEstate, error)
	GetByName(locationName string) (*types.CommercialRealEstate, error)
}

// TipRepository handles anonymous tip persistence
type TipRepository interface {
	Save(tip types.AnonymousTip) error
	GetByID(tipID string) (*types.AnonymousTip, error)
	GetRecent(limit int) ([]types.AnonymousTip, error)
	GetAll() ([]types.AnonymousTip, error)
}
