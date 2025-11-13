/*
# Module: types/error_log.go
Error log data structures including media attachments and context.

## Linked Modules
(None - types package has no dependencies)

## Tags
data-types, errors

## Exports
ErrorLog, CSpanVideo, YouTubeLivestream, TikTokVideo

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "types/error_log.go" ;
    code:description "Error log data structures including media attachments and context" ;
    code:exports :ErrorLog, :CSpanVideo, :YouTubeLivestream, :TikTokVideo ;
    code:tags "data-types", "errors" .
<!-- End LinkedDoc RDF -->
*/
package types

import "time"

// ErrorLog represents an error message with multimedia context and traceability
type ErrorLog struct {
	ID                  string    `json:"id,omitempty" dynamodbav:"id"`
	URL                 string    `json:"url,omitempty" dynamodbav:"url"` // Retrievable URL for this error log
	Message             string    `json:"message" dynamodbav:"message"`
	GifURL              string    `json:"gif_url" dynamodbav:"gif_url"` // Kept for backward compatibility
	GifURLs             []string  `json:"gif_urls,omitempty" dynamodbav:"gif_urls"` // Multiple GIFs
	Slogan              string    `json:"slogan" dynamodbav:"slogan"`
	VerboseDesc         string    `json:"verbose_desc,omitempty" dynamodbav:"verbose_desc"`
	SatiricalFix        string    `json:"satirical_fix,omitempty" dynamodbav:"satirical_fix"`
	ChildrensStory      string    `json:"childrens_story,omitempty" dynamodbav:"childrens_story"`
	SongTitle           string    `json:"song_title,omitempty" dynamodbav:"song_title"`
	SongArtist          string    `json:"song_artist,omitempty" dynamodbav:"song_artist"`
	SongURL             string    `json:"song_url,omitempty" dynamodbav:"song_url"`
	FoodImageURL        string    `json:"food_image_url,omitempty" dynamodbav:"food_image_url"`
	FoodImageAttr       string    `json:"food_image_attr,omitempty" dynamodbav:"food_image_attr"`
	MemeURL             string             `json:"meme_url,omitempty" dynamodbav:"meme_url"` // AI-generated absurdist meme
	CSpanVideo          *CSpanVideo        `json:"cspan_video,omitempty" dynamodbav:"cspan_video,omitempty"`
	CSpanLivestream     *YouTubeLivestream `json:"cspan_livestream,omitempty" dynamodbav:"cspan_livestream,omitempty"`
	TikTokVideo         *TikTokVideo       `json:"tiktok_video,omitempty" dynamodbav:"tiktok_video,omitempty"`
	UserExperienceNote  string             `json:"user_experience_note,omitempty" dynamodbav:"user_experience_note"`
	UserNoteKeywords    []string           `json:"user_note_keywords,omitempty" dynamodbav:"user_note_keywords"`
	NearbyBusinesses    []string           `json:"nearby_businesses,omitempty" dynamodbav:"nearby_businesses"`
	AnonymousTips       []string           `json:"anonymous_tips,omitempty" dynamodbav:"anonymous_tips"`
	Timestamp           time.Time          `json:"timestamp" dynamodbav:"timestamp"`

	// Rorschach test fields
	RorschachImageNumber int    `json:"rorschach_image_number,omitempty" dynamodbav:"rorschach_image_number"` // 1-10
	RorschachImageURL    string `json:"rorschach_image_url,omitempty" dynamodbav:"rorschach_image_url"`
	RorschachAIResponse  string `json:"rorschach_ai_response,omitempty" dynamodbav:"rorschach_ai_response"`       // AI interpretation
	RorschachUserResponse string `json:"rorschach_user_response,omitempty" dynamodbav:"rorschach_user_response"` // User's response

	// Traceability - links this error log back to the seed interaction that influenced its generation
	SeedInteractionType      string    `json:"seed_interaction_type,omitempty" dynamodbav:"seed_interaction_type"`
	SeedInteractionTimestamp time.Time `json:"seed_interaction_timestamp,omitempty" dynamodbav:"seed_interaction_timestamp"`
	SeedInteractionID        string    `json:"seed_interaction_id,omitempty" dynamodbav:"seed_interaction_id"`
	SeedKeywords             []string  `json:"seed_keywords,omitempty" dynamodbav:"seed_keywords"`
}

// CSpanVideo represents a C-SPAN video from search results
type CSpanVideo struct {
	Title       string `json:"title" dynamodbav:"title"`
	URL         string `json:"url" dynamodbav:"url"`
	EmbedCode   string `json:"embed_code,omitempty" dynamodbav:"embed_code"`
	Description string `json:"description,omitempty" dynamodbav:"description"`
	Date        string `json:"date,omitempty" dynamodbav:"date"`
	Duration    string `json:"duration,omitempty" dynamodbav:"duration"`
}

// YouTubeLivestream represents a C-SPAN YouTube livestream
type YouTubeLivestream struct {
	Title     string `json:"title" dynamodbav:"title"`
	VideoID   string `json:"video_id,omitempty" dynamodbav:"video_id"`
	ChannelID string `json:"channel_id,omitempty" dynamodbav:"channel_id"`
	IsLive    bool   `json:"is_live" dynamodbav:"is_live"`
}

// TikTokVideo represents a TikTok video found via Google search
type TikTokVideo struct {
	VideoID     string   `json:"video_id" dynamodbav:"video_id"`           // TikTok video ID
	URL         string   `json:"url" dynamodbav:"url"`                     // Full TikTok URL
	EmbedURL    string   `json:"embed_url" dynamodbav:"embed_url"`         // oEmbed iframe URL
	Title       string   `json:"title,omitempty" dynamodbav:"title"`       // Video title/description
	Author      string   `json:"author,omitempty" dynamodbav:"author"`     // Creator username
	AuthorURL   string   `json:"author_url,omitempty" dynamodbav:"author_url"` // Creator profile URL
	Thumbnail   string   `json:"thumbnail,omitempty" dynamodbav:"thumbnail"`   // Video thumbnail
	Tags        []string `json:"tags,omitempty" dynamodbav:"tags"`         // Hashtags used for matching
	Description string   `json:"description,omitempty" dynamodbav:"description"` // Why this video was selected
}
