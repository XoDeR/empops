package geocoder

import "context"

// Address holds fields used for geocoding.
type Address struct {
	Street     string
	City       string
	Province   string
	PostalCode string
	Country    string
	Latitude   *float64
	Longitude  *float64
}

// Coordinates is a geocoding result.
type Coordinates struct {
	Latitude  *float64
	Longitude *float64
}

// Geocoder resolves coordinates for an address.
type Geocoder interface {
	Geocode(ctx context.Context, addr Address) (Coordinates, error)
}

// Noop returns manual coordinates when provided.
type Noop struct{}

func (Noop) Geocode(_ context.Context, addr Address) (Coordinates, error) {
	return Coordinates{Latitude: addr.Latitude, Longitude: addr.Longitude}, nil
}
