package panchangam

import (
	"time"

	"github.com/purohit-platform/backend/internal/models"
)

var rahuKalamWindows = map[time.Weekday]string{
	time.Sunday:    "04:30 PM - 06:00 PM",
	time.Monday:    "07:30 AM - 09:00 AM",
	time.Tuesday:   "03:00 PM - 04:30 PM",
	time.Wednesday: "12:00 PM - 01:30 PM",
	time.Thursday:  "01:30 PM - 03:00 PM",
	time.Friday:    "10:30 AM - 12:00 PM",
	time.Saturday:  "09:00 AM - 10:30 AM",
}

var yamagandamWindows = map[time.Weekday]string{
	time.Sunday:    "12:00 PM - 01:30 PM",
	time.Monday:    "10:30 AM - 12:00 PM",
	time.Tuesday:   "09:00 AM - 10:30 AM",
	time.Wednesday: "07:30 AM - 09:00 AM",
	time.Thursday:  "06:00 AM - 07:30 AM",
	time.Friday:    "03:00 PM - 04:30 PM",
	time.Saturday:  "01:30 PM - 03:00 PM",
}

var nakshatras = []string{
	"Ashwini", "Bharani", "Krittika", "Rohini", "Mrigashira", "Ardra",
	"Punarvasu", "Pushya", "Ashlesha", "Magha", "Purva Phalguni", "Uttara Phalguni",
	"Hasta", "Chitra", "Swati", "Vishakha", "Anuradha", "Jyeshtha",
	"Mula", "Purva Ashadha", "Uttara Ashadha", "Shravana", "Dhanishta",
	"Shatabhisha", "Purva Bhadrapada", "Uttara Bhadrapada", "Revati",
}

// ComputePanchangam generates the daily astrological markers
func ComputePanchangam(t time.Time) models.PanchangamDay {
	dayOfYear := t.YearDay()
	nakshatra := nakshatras[dayOfYear%len(nakshatras)]

	tithiNum := (dayOfYear % 15) + 1
	paksha := "Shukla"
	if (dayOfYear/15)%2 == 1 {
		paksha = "Krishna"
	}

	tithiName := getTithiName(tithiNum, paksha)
	weekday := t.Weekday()

	return models.PanchangamDay{
		Date:          t,
		Tithi:         tithiName,
		Nakshatra:     nakshatra,
		Yoga:          "Siddhi Yoga",
		Karana:        "Bava",
		RahuKalam:     rahuKalamWindows[weekday],
		Yamagandam:    yamagandamWindows[weekday],
		GulikaKalam:   "07:30 AM - 09:00 AM",
		AuspiciousDay: weekday != time.Tuesday && weekday != time.Saturday,
	}
}

func getTithiName(tithi int, paksha string) string {
	names := []string{"", "Prathama", "Dwitiya", "Tritiya", "Chaturthi", "Panchami",
		"Shashti", "Saptami", "Ashtami", "Navami", "Dashami",
		"Ekadashi", "Dwadashi", "Trayodashi", "Chaturdashi", "Purnima/Amavasya"}
	return paksha + " Paksha " + names[tithi]
}
