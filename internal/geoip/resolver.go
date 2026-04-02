package geoip

import (
	"net"

	"github.com/oschwald/maxminddb-golang"
)

type Location struct {
	Country string
	City    string
}

type Resolver struct {
	db *maxminddb.Reader
}

type geoRecord struct {
	Country struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	City struct {
		Names map[string]string `maxminddb:"city"`
	} `maxminddb:"city"`
}

func NewResolver(dbPath string) (*Resolver, error) {
	db, err := maxminddb.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return &Resolver{db: db}, nil
}

func (r *Resolver) Lookup(ipStr string) *Location {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return &Location{}
	}
	var record geoRecord
	if err := r.db.Lookup(ip, &record); err != nil {
		return &Location{}
	}
	return &Location{
		Country: record.Country.Names["en"],
		City:    record.City.Names["en"],
	}
}

func (r *Resolver) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
