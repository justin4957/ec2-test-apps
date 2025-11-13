/*
# Module: types/tip.go
Anonymous tip submission data structures.

## Linked Modules
(None - types package has no dependencies)

## Tags
data-types, tips

## Exports
AnonymousTip

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "types/tip.go" ;
    code:description "Anonymous tip submission data structures" ;
    code:exports :AnonymousTip ;
    code:tags "data-types", "tips" .
<!-- End LinkedDoc RDF -->
*/
package types

import "time"

// AnonymousTip represents an anonymous tip submission with moderation metadata
type AnonymousTip struct {
	ID               string    `json:"id" dynamodbav:"id"`
	TipContent       string    `json:"tip_content" dynamodbav:"tip_content"`
	ModeratedContent string    `json:"moderated_content" dynamodbav:"moderated_content"`
	UserHash         string    `json:"user_hash" dynamodbav:"user_hash"`
	UserMetadata     string    `json:"user_metadata" dynamodbav:"user_metadata"`
	ModerationStatus string    `json:"moderation_status" dynamodbav:"moderation_status"`
	ModerationReason string    `json:"moderation_reason,omitempty" dynamodbav:"moderation_reason"`
	Keywords         []string  `json:"keywords,omitempty" dynamodbav:"keywords"`
	Timestamp        time.Time `json:"timestamp" dynamodbav:"timestamp"`
	IPAddress        string    `json:"ip_address,omitempty" dynamodbav:"ip_address"`
}
