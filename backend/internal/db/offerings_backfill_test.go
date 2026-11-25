package db

import (
	"os"
	"reflect"
	"testing"

	"github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/models"
)

func TestBackfillPriestOfferings(t *testing.T) {
	storage, _, _ := createIsolatedAuthTestDB(t)
	priestID, err := storage.RegisterPriest(models.PriestRegistrationRequest{Phone: "+919876540498", FullName: "Backfill Test", Password: "Password123!", Sampradaya: "Smartha", VedaShakha: "Rigveda", ExperienceYears: 5, City: "Hyderabad", ConsentAgreed: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = storage.DB.Exec(`UPDATE priests SET services_offered=ARRAY['Existing Service'] WHERE id=$1`, priestID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = storage.DB.Exec(`INSERT INTO priest_custom_services (priest_id, title, category, dakshina_fee, duration_minutes, is_active) VALUES
      ($1, ' New Ritual ', 'Puja', 0, 60, true), ($1, 'new ritual', 'Puja', 0, 60, true),
      ($1, 'existing service', 'Puja', 0, 60, true), ($1, 'Inactive Ritual', 'Puja', 0, 60, false)`, priestID)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/000019_backfill_priest_offerings.sql")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = storage.DB.Exec(string(migration)); err != nil {
			t.Fatal(err)
		}
	}
	var got pq.StringArray
	if err = storage.DB.QueryRow(`SELECT services_offered FROM priests WHERE id=$1`, priestID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want := (pq.StringArray{"Existing Service", "New Ritual"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
