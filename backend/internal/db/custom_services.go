package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/purohit-platform/backend/internal/models"
)

// AutoTranslateVedic generates authentic Telugu and Hindi equivalents for ritual titles
func AutoTranslateVedic(title string) (titleTe string, titleHi string) {
	lower := strings.ToLower(title)
	if strings.Contains(lower, "satyanarayana") {
		return "శ్రీ సత్యనారాయణ స్వామి వ్రతం", "श्री सत्यनारायण स्वामी व्रत कथा"
	}
	if strings.Contains(lower, "gruhapravesam") || strings.Contains(lower, "house warming") {
		return "గృహప్రవేశం (నూతన గృహ ప్రవేశ మహోత్సవం)", "गृहप्रवेश पूजा एवं वास्तु शांति हवन"
	}
	if strings.Contains(lower, "ganapathi") || strings.Contains(lower, "ganesha") {
		return "మహా గణపతి హోమం / యాగానుష్ఠానం", "महा गणपति हवन / विघ्नहर्ता अनुष्ठान"
	}
	if strings.Contains(lower, "namakaranam") || strings.Contains(lower, "ayushya") {
		return "నామకరణం & ఆయుష్య హోమం", "नामकरण संस्कार एवं आयुष्य हवन"
	}
	if strings.Contains(lower, "mrityunjaya") {
		return "మహా మృత్యుంజయ హోమం / ఆయుర్వృద్ధి శాంతి", "महा मृत्युंजय हवन / आरोग्य शांति अनुष्ठान"
	}
	if strings.Contains(lower, "navagraha") && strings.Contains(lower, "dhanvantari") {
		return "నవగ్రహ + ధన్వంతరి మహా హోమం (ప్రత్యేక పౌరోహిత్యం)", "नवग्रह एवं धन्वंतरि महाहवन विशेष अनुष्ठान"
	}
	if strings.Contains(lower, "navagraha") {
		return "నవగ్రహ శాంతి హోమం & దోష నివారణ", "नवग्रह शांति हवन एवं ग्रह दोष निवारण"
	}
	if strings.Contains(lower, "lalitha") || strings.Contains(lower, "chandi") || strings.Contains(lower, "kumkuma") {
		return "శ్రీ లలితా సహస్రనామ కుంకుమార్చన + చండీ పారాయణం", "श्री ललिता सहस्रनाम कुमकुमार्चन एवं चंडी पाठ"
	}
	if strings.Contains(lower, "sudarshana") {
		return "శ్రీ సుదర్శన నారసింహ మహా హోమం", "श्री सुदर्शन नृसिंह महायज्ञ"
	}
	if strings.Contains(lower, "rudrabhishekam") || strings.Contains(lower, "rudra") {
		return "మహా రుద్రాభిషేకం & ఏకాదశ రుద్ర జపం", "महा रुद्राभिषेक एवं एकादश रुद्र जप"
	}

	// Default fallback with Vedic prefixes
	return fmt.Sprintf("శ్రీ %s (పౌరోహిత్య సేవ)", title), fmt.Sprintf("श्री %s (वैदिक अनुष्ठान)", title)
}

// GetPriestCustomServices retrieves custom pooja packages created by a priest, localized to selected language
func (s *Storage) GetPriestCustomServices(priestID string, lang string) ([]models.PriestCustomService, error) {
	query := `
		SELECT id, COALESCE(priest_id::text, ''), title, 
		       COALESCE(title_en, title) as t_en, 
		       COALESCE(title_te, title) as t_te, 
		       COALESCE(title_hi, title) as t_hi, 
		       category, dakshina_fee, samagri_kit_price, 
		       duration_minutes, is_online_eligible, samagri_checklist, 
		       description, 
		       COALESCE(description_en, description) as d_en, 
		       COALESCE(description_te, description) as d_te, 
		       COALESCE(description_hi, description) as d_hi, 
		       is_active, created_at 
		FROM priest_custom_services 
		WHERE is_active = TRUE `

	var rows *sql.Rows
	var err error
	if priestID != "" && priestID != "all" {
		query += ` AND priest_id = $1 ORDER BY created_at DESC`
		rows, err = s.DB.Query(query, priestID)
	} else {
		query += ` ORDER BY created_at DESC`
		rows, err = s.DB.Query(query)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query custom services: %w", err)
	}
	defer rows.Close()

	var services []models.PriestCustomService
	for rows.Next() {
		var cs models.PriestCustomService
		var rawTitle, rawDesc string
		var checklistJSON []byte
		if err := rows.Scan(
			&cs.ID, &cs.PriestID, &rawTitle,
			&cs.TitleEn, &cs.TitleTe, &cs.TitleHi,
			&cs.Category, &cs.DakshinaFee,
			&cs.SamagriKitPrice, &cs.DurationMinutes, &cs.IsOnlineEligible,
			&checklistJSON, &rawDesc,
			&cs.DescriptionEn, &cs.DescriptionTe, &cs.DescriptionHi,
			&cs.IsActive, &cs.CreatedAt,
		); err != nil {
			continue
		}
		if len(checklistJSON) > 0 {
			_ = json.Unmarshal(checklistJSON, &cs.SamagriChecklist)
		}

		displayTitle := rawTitle
		displayDesc := rawDesc
		switch strings.ToLower(lang) {
		case "te", "telugu":
			if cs.TitleTe != "" {
				displayTitle = cs.TitleTe
			}
			if cs.DescriptionTe != "" {
				displayDesc = cs.DescriptionTe
			}
		case "hi", "hindi":
			if cs.TitleHi != "" {
				displayTitle = cs.TitleHi
			}
			if cs.DescriptionHi != "" {
				displayDesc = cs.DescriptionHi
			}
		case "en", "english":
			if cs.TitleEn != "" {
				displayTitle = cs.TitleEn
			}
			if cs.DescriptionEn != "" {
				displayDesc = cs.DescriptionEn
			}
		}

		cs.Title = displayTitle
		cs.Description = displayDesc
		services = append(services, cs)
	}
	return services, nil
}

// CreatePriestCustomService creates a new customized ritual offering with Dakshina pricing and 3-language prep
func (s *Storage) CreatePriestCustomService(req models.CreateCustomServiceRequest) (string, error) {
	checklistJSON, err := json.Marshal(req.SamagriChecklist)
	if err != nil {
		checklistJSON = []byte("[]")
	}

	// If priest didn't provide translations, auto-translate into Telugu and Hindi
	titleEn := req.TitleEn
	if titleEn == "" {
		titleEn = req.Title
	}

	titleTe := req.TitleTe
	titleHi := req.TitleHi
	if titleTe == "" || titleHi == "" {
		autoTe, autoHi := AutoTranslateVedic(req.Title)
		if titleTe == "" {
			titleTe = autoTe
		}
		if titleHi == "" {
			titleHi = autoHi
		}
	}

	descEn := req.DescriptionEn
	if descEn == "" {
		descEn = req.Description
	}
	descTe := req.DescriptionTe
	if descTe == "" {
		descTe = fmt.Sprintf("పురోహితులు నిర్వహించే పవిత్ర వేదోక్త పూజా విధానము: %s", req.Description)
	}
	descHi := req.DescriptionHi
	if descHi == "" {
		descHi = fmt.Sprintf("पुरोहित जी द्वारा संपन्न की जाने वाली पावन वैदिक पूजा: %s", req.Description)
	}

	var validPriest bool
	if req.PriestID != "" {
		var pid string
		if err := s.DB.QueryRow(`SELECT id::text FROM priests WHERE id::text = $1`, req.PriestID).Scan(&pid); err == nil && pid != "" {
			validPriest = true
			req.PriestID = pid
		}
	}
	if !validPriest {
		s.DB.QueryRow(`SELECT id::text FROM priests LIMIT 1`).Scan(&req.PriestID)
	}

	query := `
		INSERT INTO priest_custom_services (
			priest_id, title, title_en, title_te, title_hi, category, dakshina_fee, samagri_kit_price, 
			duration_minutes, is_online_eligible, samagri_checklist, description, description_en, description_te, description_hi
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) 
		RETURNING id
	`
	var id string
	err = s.DB.QueryRow(
		query, req.PriestID, req.Title, titleEn, titleTe, titleHi, req.Category, req.DakshinaFee,
		req.SamagriKitPrice, req.DurationMinutes, req.IsOnlineEligible,
		checklistJSON, req.Description, descEn, descTe, descHi,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to insert custom service: %w", err)
	}
	return id, nil
}

// GetAllServiceTranslations returns all pooja_catalog and priest_custom_services with 3-language fields for CMS
func (s *Storage) GetAllServiceTranslations() ([]models.ServiceTranslationItem, error) {
	var results []models.ServiceTranslationItem

	// 1. Fetch pooja_catalog
	catQuery := `
		SELECT id, title, 
		       COALESCE(title_en, title), COALESCE(title_te, title), COALESCE(title_hi, title),
		       COALESCE(description_en, description), COALESCE(description_te, description), COALESCE(description_hi, description),
		       category, base_price
		FROM pooja_catalog
		ORDER BY base_price ASC
	`
	catRows, err := s.DB.Query(catQuery)
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var item models.ServiceTranslationItem
			item.Type = "CATALOG"
			item.PriestName = "Temple Central Catalog"
			if err := catRows.Scan(
				&item.ID, &item.OriginalTitle,
				&item.TitleEn, &item.TitleTe, &item.TitleHi,
				&item.DescriptionEn, &item.DescriptionTe, &item.DescriptionHi,
				&item.Category, &item.BasePrice,
			); err == nil {
				results = append(results, item)
			}
		}
	}

	// 2. Fetch priest_custom_services
	customQuery := `
		SELECT cs.id, COALESCE(cs.priest_id::text, ''), COALESCE(p.full_name, 'Purohit'), cs.title,
		       COALESCE(cs.title_en, cs.title), COALESCE(cs.title_te, cs.title), COALESCE(cs.title_hi, cs.title),
		       COALESCE(cs.description_en, cs.description), COALESCE(cs.description_te, cs.description), COALESCE(cs.description_hi, cs.description),
		       cs.category, cs.dakshina_fee
		FROM priest_custom_services cs
		LEFT JOIN priests p ON cs.priest_id = p.id
		WHERE cs.is_active = TRUE
		ORDER BY cs.created_at DESC
	`
	customRows, err := s.DB.Query(customQuery)
	if err == nil {
		defer customRows.Close()
		for customRows.Next() {
			var item models.ServiceTranslationItem
			item.Type = "PRIEST_CUSTOM"
			if err := customRows.Scan(
				&item.ID, &item.PriestID, &item.PriestName, &item.OriginalTitle,
				&item.TitleEn, &item.TitleTe, &item.TitleHi,
				&item.DescriptionEn, &item.DescriptionTe, &item.DescriptionHi,
				&item.Category, &item.BasePrice,
			); err == nil {
				results = append(results, item)
			}
		}
	}

	return results, nil
}

// UpdateServiceTranslation updates 3-language fields for a service from CMS
func (s *Storage) UpdateServiceTranslation(req models.UpdateServiceTranslationRequest) error {
	if req.Type == "CATALOG" {
		query := `
			UPDATE pooja_catalog 
			SET title_en = $2, title_te = $3, title_hi = $4,
			    description_en = $5, description_te = $6, description_hi = $7
			WHERE id = $1
		`
		_, err := s.DB.Exec(query, req.ID, req.TitleEn, req.TitleTe, req.TitleHi, req.DescriptionEn, req.DescriptionTe, req.DescriptionHi)
		return err
	}

	query := `
		UPDATE priest_custom_services 
		SET title_en = $2, title_te = $3, title_hi = $4,
		    description_en = $5, description_te = $6, description_hi = $7
		WHERE id = $1
	`
	_, err := s.DB.Exec(query, req.ID, req.TitleEn, req.TitleTe, req.TitleHi, req.DescriptionEn, req.DescriptionTe, req.DescriptionHi)
	return err
}

// AutoTranslateAllServices scans and auto-populates Telugu and Hindi translations for untranslated services
func (s *Storage) AutoTranslateAllServices() (int, error) {
	count := 0

	// 1. Check custom services where title_te is equal to title or empty
	rows, err := s.DB.Query(`SELECT id, title FROM priest_custom_services WHERE title_te IS NULL OR title_te = title OR title_te = ''`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, title string
			if err := rows.Scan(&id, &title); err == nil {
				te, hi := AutoTranslateVedic(title)
				s.DB.Exec(`
					UPDATE priest_custom_services 
					SET title_en = COALESCE(NULLIF(title_en, ''), $2),
					    title_te = $3, 
					    title_hi = $4 
					WHERE id = $1
				`, id, title, te, hi)
				count++
			}
		}
	}

	// 2. Check catalog items
	catRows, err := s.DB.Query(`SELECT id, title FROM pooja_catalog WHERE title_te IS NULL OR title_te = title OR title_te = ''`)
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var id, title string
			if err := catRows.Scan(&id, &title); err == nil {
				te, hi := AutoTranslateVedic(title)
				s.DB.Exec(`
					UPDATE pooja_catalog 
					SET title_en = COALESCE(NULLIF(title_en, ''), $2),
					    title_te = $3, 
					    title_hi = $4 
					WHERE id = $1
				`, id, title, te, hi)
				count++
			}
		}
	}

	return count, nil
}
