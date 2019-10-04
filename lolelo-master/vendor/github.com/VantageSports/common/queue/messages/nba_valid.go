package messages

import "fmt"

func (g *GameFileMigration) Valid() error {
	if g.GameId == "" || g.Source == "" || g.Destination == "" {
		return fmt.Errorf("game_id, source, and destination are required")
	}
	return nil
}

func (m MigrateChances) Valid() error {
	if m.SourceSchema == "" || m.DestSchema == "" || m.GameMigration == nil {
		return fmt.Errorf("source_schema dest_schema, and game_migration are required")
	}
	return m.GameMigration.Valid()
}

func (te *OCRTextExtraction) Valid() error {
	if te.GameId == "" || te.VideoPath == "" || te.OutputImagesPath == "" || te.ExtractedFramesPerSecond <= 0 {
		return fmt.Errorf("game_id, video_path, output_images_path, and eps are all required")
	}
	return te.Dimensions.Valid()
}

func (d *OCRDimensions) Valid() error {
	if d.Width <= 0 || d.Height <= 0 {
		return fmt.Errorf("width and height need to be larger than 0")
	}
	if d.X < 0 || d.Y < 0 {
		return fmt.Errorf("x and y need to be zero or larger")
	}
	if d.SlotWidth < 0 || d.SlotHeight < 0 {
		return fmt.Errorf("slot_width and slot_height need to be zero or larger")
	}
	return nil
}

const (
	V_ALL          = "all"          // Perform all validations.
	V_META         = "meta"         // Meta-integrity.
	V_CHANCE_MOVE  = "chance_move"  // Chance/Move agreement.
	V_CROSS_MOVE   = "cross_move"   // Move/Move (corroborative) agreement.
	V_CROSS_CHANCE = "cross_chance" // Chance agreement.
	V_MOVE_SCHEMA  = "move_schema"  // Move schema adherence.
)

func (v *ValidateChances) Valid() error {
	if err := v.GameMigration.Valid(); err != nil {
		return err
	}
	if len(v.Validations) == 0 {
		return fmt.Errorf("at least one validation (or 'all') is required")
	}
	for _, each := range v.Validations {
		switch each {
		case V_ALL, V_META, V_CHANCE_MOVE, V_CROSS_MOVE, V_CROSS_CHANCE:
			continue
		default:
			return fmt.Errorf("unrecognized validation: %s", each)
		}
	}
	return nil
}

func (s *VideoSplit) Valid() error {
	if len(s.Segments) == 0 {
		return fmt.Errorf("len(segments) == 0")
	}
	if len(s.Destinations) == 0 {
		return fmt.Errorf("len(destinations) == 0")
	}
	if len(s.Segments) != len(s.Destinations) {
		return fmt.Errorf("len(segments) != len(destinations)")
	}
	if s.Source == "" {
		return fmt.Errorf("source == \"\"")
	}
	for i, d := range s.Destinations {
		if d == "" {
			return fmt.Errorf("destinations[%d] == \"\"", i)
		}
	}
	return nil
}
