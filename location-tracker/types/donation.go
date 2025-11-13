/*
# Module: types/donation.go
Stripe donation and payment data structures.

## Linked Modules
(None - types package has no dependencies)

## Tags
data-types, payments, stripe

## Exports
Donation

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "types/donation.go" ;
    code:description "Stripe donation and payment data structures" ;
    code:exports :Donation ;
    code:tags "data-types", "payments", "stripe" .
<!-- End LinkedDoc RDF -->
*/
package types

import "time"

// Donation represents a Stripe donation record with metadata
type Donation struct {
	ID                string    `json:"id" dynamodbav:"id"`
	DonationType      string    `json:"donation_type" dynamodbav:"donation_type"` // "meme_disclaimer" or "church_committee"
	Amount            int64     `json:"amount" dynamodbav:"amount"`               // Amount in cents
	StripePaymentID   string    `json:"stripe_payment_id" dynamodbav:"stripe_payment_id"`
	UserHash          string    `json:"user_hash,omitempty" dynamodbav:"user_hash"`
	Timestamp         time.Time `json:"timestamp" dynamodbav:"timestamp"`
	IPAddress         string    `json:"ip_address,omitempty" dynamodbav:"ip_address"`
	Status            string    `json:"status" dynamodbav:"status"` // "pending", "succeeded", "failed"
	BankRecordPurpose string    `json:"bank_record_purpose" dynamodbav:"bank_record_purpose"`
}
