package engine

// TrackSegment pist segmenti tanımı
type TrackSegment struct {
	StartPos    float64 // Başlangıç pozisyonu (metre)
	EndPos      float64 // Bitiş pozisyonu (metre)
	Type        string  // "straight", "corner_light", "corner_medium", "corner_hard", "hairpin"
	MaxSpeed    float64 // Bu segmentteki maksimum güvenli hız (km/h)
	CornerAngle float64 // Viraj açısı (derece, düzlük için 0)
	Name        string  // Segment adı
}

// SilesiaRingTrack - Silesia Ring pist verileri (4200m)
// Gerçek pist layout'una yakın simülasyon
var SilesiaRingTrack = []TrackSegment{
	// Start/Finish düzlüğü
	{StartPos: 0, EndPos: 350, Type: "straight", MaxSpeed: 40, CornerAngle: 0, Name: "Start/Finish Straight"},

	// Turn 1 - Sağ viraj (hafif)
	{StartPos: 350, EndPos: 500, Type: "corner_light", MaxSpeed: 32, CornerAngle: 45, Name: "Turn 1"},

	// Kısa düzlük
	{StartPos: 500, EndPos: 700, Type: "straight", MaxSpeed: 38, CornerAngle: 0, Name: "Back Straight 1"},

	// Turn 2-3 - S virajı
	{StartPos: 700, EndPos: 850, Type: "corner_medium", MaxSpeed: 28, CornerAngle: 60, Name: "Turn 2"},
	{StartPos: 850, EndPos: 1000, Type: "corner_medium", MaxSpeed: 28, CornerAngle: -55, Name: "Turn 3"},

	// Orta düzlük
	{StartPos: 1000, EndPos: 1350, Type: "straight", MaxSpeed: 40, CornerAngle: 0, Name: "Middle Straight"},

	// Turn 4 - Hairpin (sert viraj)
	{StartPos: 1350, EndPos: 1550, Type: "hairpin", MaxSpeed: 18, CornerAngle: 180, Name: "Hairpin Turn 4"},

	// Kısa düzlük
	{StartPos: 1550, EndPos: 1750, Type: "straight", MaxSpeed: 35, CornerAngle: 0, Name: "Short Straight"},

	// Turn 5 - Sol viraj
	{StartPos: 1750, EndPos: 1900, Type: "corner_medium", MaxSpeed: 25, CornerAngle: -75, Name: "Turn 5"},

	// Turn 6-7 - Teknik bölüm
	{StartPos: 1900, EndPos: 2050, Type: "corner_light", MaxSpeed: 30, CornerAngle: 40, Name: "Turn 6"},
	{StartPos: 2050, EndPos: 2200, Type: "corner_light", MaxSpeed: 30, CornerAngle: -35, Name: "Turn 7"},

	// Arka düzlük (en uzun)
	{StartPos: 2200, EndPos: 2800, Type: "straight", MaxSpeed: 40, CornerAngle: 0, Name: "Back Straight"},

	// Turn 8 - Yüksek hızlı viraj
	{StartPos: 2800, EndPos: 3000, Type: "corner_light", MaxSpeed: 33, CornerAngle: 50, Name: "Turn 8"},

	// Düzlük
	{StartPos: 3000, EndPos: 3300, Type: "straight", MaxSpeed: 38, CornerAngle: 0, Name: "Straight 9"},

	// Turn 9-10 - Şikan
	{StartPos: 3300, EndPos: 3450, Type: "corner_hard", MaxSpeed: 22, CornerAngle: 90, Name: "Turn 9 Chicane"},
	{StartPos: 3450, EndPos: 3600, Type: "corner_hard", MaxSpeed: 22, CornerAngle: -85, Name: "Turn 10 Chicane"},

	// Son düzlük
	{StartPos: 3600, EndPos: 3850, Type: "straight", MaxSpeed: 38, CornerAngle: 0, Name: "Pre-Final Straight"},

	// Son viraj
	{StartPos: 3850, EndPos: 4050, Type: "corner_medium", MaxSpeed: 26, CornerAngle: 70, Name: "Final Turn"},

	// Finish düzlüğü
	{StartPos: 4050, EndPos: 4200, Type: "straight", MaxSpeed: 40, CornerAngle: 0, Name: "Finish Straight"},
}

const TrackLength = 4200.0 // metre

// GetSegmentAtPosition belirtilen pozisyondaki segment'i döndürür
func GetSegmentAtPosition(position float64) *TrackSegment {
	// Pozisyonu normalize et (0-4200 arası)
	pos := position
	for pos >= TrackLength {
		pos -= TrackLength
	}
	for pos < 0 {
		pos += TrackLength
	}

	for i := range SilesiaRingTrack {
		if pos >= SilesiaRingTrack[i].StartPos && pos < SilesiaRingTrack[i].EndPos {
			return &SilesiaRingTrack[i]
		}
	}
	// Fallback
	return &SilesiaRingTrack[0]
}

// GetUpcomingSegments önümüzdeki N metredeki segmentleri döndürür
func GetUpcomingSegments(position float64, lookAheadMeters float64) []TrackSegment {
	var segments []TrackSegment
	pos := position

	for pos >= TrackLength {
		pos -= TrackLength
	}

	endPos := pos + lookAheadMeters
	for _, seg := range SilesiaRingTrack {
		segStart := seg.StartPos
		segEnd := seg.EndPos

		// Segment görüş alanında mı?
		if segEnd > pos && segStart < endPos {
			segments = append(segments, seg)
		}
		// Tur sonu geçişi
		if endPos > TrackLength {
			wrappedEnd := endPos - TrackLength
			if segStart < wrappedEnd {
				segments = append(segments, seg)
			}
		}
	}
	return segments
}

// GetMinSpeedAhead önümüzdeki N metredeki minimum hız limitini döndürür
func GetMinSpeedAhead(position float64, lookAheadMeters float64) float64 {
	segments := GetUpcomingSegments(position, lookAheadMeters)
	if len(segments) == 0 {
		return 40.0 // Default max
	}

	minSpeed := 100.0
	for _, seg := range segments {
		if seg.MaxSpeed < minSpeed {
			minSpeed = seg.MaxSpeed
		}
	}
	return minSpeed
}
