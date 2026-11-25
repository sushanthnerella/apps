package db

import (
	"github.com/purohit-platform/backend/internal/models"
	"testing"
)

func TestCustomOfferingMakesPriestDiscoverable(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)
	priestID, err := storage.RegisterPriest(models.PriestRegistrationRequest{Phone: "+919876540499", FullName: "Offering Test", Password: "Password123!", Sampradaya: "Smartha", VedaShakha: "Rigveda", ExperienceYears: 5, City: "Hyderabad", ConsentAgreed: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = storage.DB.Exec(`UPDATE priests SET is_online=true, location_confirmed=true, latitude=17.385, longitude=78.4867, services_offered=ARRAY['Existing Service'] WHERE id=$1`, priestID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = storage.CreatePriestCustomService(models.CreateCustomServiceRequest{PriestID: priestID, Title: "New CMS Ritual", DurationMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	priests, err := storage.GetNearbyPriests(17.385, 78.4867, 1000, "New CMS Ritual")
	if err != nil {
		t.Fatal(err)
	}
	if len(priests) != 1 || priests[0].ID != priestID {
		t.Fatalf("new offering should find its priest: %+v", priests)
	}
	priests, err = storage.GetNearbyPriests(17.385, 78.4867, 1000, "Existing Service")
	if err != nil || len(priests) != 1 {
		t.Fatalf("existing offering should remain searchable: %+v, %v", priests, err)
	}
}
