# Google Places API (New) Technical Specification & Integration Guide

## 1. Authentication Mechanisms

Google Maps Platform (Places API New) supports two primary authentication mechanisms for RESTful requests:

### 1.1. API Key via Custom HTTP Header (Recommended & Used in This Service)
- **Header**: `X-Goog-Api-Key: <YOUR_API_KEY>`
- **Why this is recommended**:
  1. **Security**: Keeps your API key out of server access logs, browser history, and proxy query strings.
  2. **Performance**: Standard static header, zero OAuth token exchange round-trips.
  3. **Cloud Console Restrictions**: Allows restricting key usage to server IP addresses or specific API scopes (`Places API (New)`).

```http
POST /v1/places:searchText HTTP/1.1
Host: places.googleapis.com
Content-Type: application/json
X-Goog-Api-Key: AIzaSyD-YOUR-GOOGLE-API-KEY
X-Goog-FieldMask: places.id,places.displayName,places.formattedAddress,places.websiteUri,places.nationalPhoneNumber,places.rating,nextPageToken
```

### 1.2. API Key via URL Query Parameter (Alternative / Fallback)
- **URL**: `https://places.googleapis.com/v1/places:searchText?key=<YOUR_API_KEY>`
- **Note**: Functional, but headers are preferred for security best practices.

### 1.3. OAuth 2.0 Bearer Token (Google Cloud Service Account)
- **Header**: `Authorization: Bearer <OAUTH_ACCESS_TOKEN>`
- **Scope**: `https://www.googleapis.com/auth/cloud-platform` or `https://www.googleapis.com/auth/maps-platform.places`
- **Use case**: Enterprise backend microservices already authenticating via GCP IAM / Workload Identity. For standard standalone or self-hosted CLI/dashboards, API Key is standard.

---

## 2. API Endpoints Specification

### 2.1. Text Search (New) — `places:searchText` (Primary Extraction Endpoint)

Used for keyword and category discovery across a city or geographic boundary.

- **Method**: `POST`
- **URL**: `https://places.googleapis.com/v1/places:searchText`
- **Headers**:
  ```http
  Content-Type: application/json
  X-Goog-Api-Key: <API_KEY>
  X-Goog-FieldMask: places.id,places.displayName,places.formattedAddress,places.location,places.types,places.primaryType,places.websiteUri,places.nationalPhoneNumber,places.rating,places.userRatingCount,places.businessStatus,places.googleMapsUri,places.regularOpeningHours,places.priceLevel,nextPageToken
  ```

#### Request Body Schema (JSON):
```json
{
  "textQuery": "specialty coffee in Seattle WA",
  "includedType": "coffee_shop",
  "pageSize": 20,
  "minRating": 3.5,
  "openNow": false,
  "priceLevels": ["PRICE_LEVEL_MODERATE", "PRICE_LEVEL_EXPENSIVE"],
  "locationBias": {
    "circle": {
      "center": {
        "latitude": 47.6062,
        "longitude": -122.3321
      },
      "radius": 5000.0
    }
  },
  "pageToken": "" 
}
```

#### Response Payload Schema (JSON):
```json
{
  "places": [
    {
      "id": "ChIJN1t_tDeuEmsRUsoyG83frY4",
      "types": ["coffee_shop", "cafe", "food", "point_of_interest", "establishment"],
      "formattedAddress": "1912 Pike Pl, Seattle, WA 98101, USA",
      "location": {
        "latitude": 47.6097,
        "longitude": -122.3421
      },
      "rating": 4.7,
      "googleMapsUri": "https://maps.google.com/?cid=1028111959637413554",
      "websiteUri": "https://www.starbucks.com/store-locator/store/11674/pike-place-1912-pike-place-seattle-wa-981011013-us",
      "regularOpeningHours": {
        "openNow": true,
        "periods": [
          {
            "open": { "day": 1, "hour": 6, "minute": 0 },
            "close": { "day": 1, "hour": 20, "minute": 0 }
          }
        ],
        "weekdayDescriptions": [
          "Monday: 6:00 AM – 8:00 PM",
          "Tuesday: 6:00 AM – 8:00 PM"
        ]
      },
      "businessStatus": "OPERATIONAL",
      "priceLevel": "PRICE_LEVEL_MODERATE",
      "userRatingCount": 18450,
      "displayName": {
        "text": "Pike Place Starbucks",
        "languageCode": "en"
      },
      "primaryType": "coffee_shop",
      "nationalPhoneNumber": "(206) 448-8762"
    }
  ],
  "nextPageToken": "AUc7tXX1..."
}
```

---

### 2.2. Nearby Search (New) — `places:searchNearby`

Used when scanning an exact radius coordinate grid without a text query.

- **Method**: `POST`
- **URL**: `https://places.googleapis.com/v1/places:searchNearby`
- **Headers**:
  ```http
  Content-Type: application/json
  X-Goog-Api-Key: <API_KEY>
  X-Goog-FieldMask: places.id,places.displayName,places.formattedAddress,places.location,places.types,places.websiteUri,places.nationalPhoneNumber,places.rating,places.userRatingCount,places.googleMapsUri
  ```

#### Request Body Schema (JSON):
```json
{
  "includedTypes": ["restaurant", "bar"],
  "maxResultCount": 20,
  "locationRestriction": {
    "circle": {
      "center": {
        "latitude": 40.7128,
        "longitude": -74.0060
      },
      "radius": 1500.0
    }
  },
  "rankPreference": "DISTANCE"
}
```

---

### 2.3. Place Details (New) — `places/{placeId}`

Used when looking up deep details or refreshing a specific place by Place ID.

- **Method**: `GET`
- **URL**: `https://places.googleapis.com/v1/places/{placeId}`
- **Headers**:
  ```http
  X-Goog-Api-Key: <API_KEY>
  X-Goog-FieldMask: id,displayName,formattedAddress,location,websiteUri,nationalPhoneNumber,rating,userRatingCount,regularOpeningHours,photos,reviews
  ```

---

## 3. Field Masks & Pricing SKU Matrix

Google Maps Platform calculates charges based on the **highest-tier field** included in `X-Goog-FieldMask`:

```
┌───────────────────────────────────────────────────────────────────────────────────────┐
│ SKU TIER 1: ESSENTIALS (Free: 10,000 calls/mo | Overage: $5.00 / 1,000)               │
│ - places.id                                                                           │
│ - places.displayName                                                                  │
│ - places.formattedAddress, shortFormattedAddress, addressComponents                   │
│ - places.location (lat/lng), viewport, plusCode                                       │
│ - places.types, primaryType, primaryTypeDisplayName                                   │
├───────────────────────────────────────────────────────────────────────────────────────┤
│ SKU TIER 2: PRO (Free: 5,000 calls/mo | Overage: $17.00 - $25.00 / 1,000)             │
│ - places.websiteUri                                                                   │
│ - places.nationalPhoneNumber, internationalPhoneNumber                                │
│ - places.rating, userRatingCount                                                      │
│ - places.businessStatus                                                               │
│ - places.priceLevel                                                                   │
│ - places.regularOpeningHours, currentOpeningHours                                     │
│ - places.googleMapsUri                                                                │
├───────────────────────────────────────────────────────────────────────────────────────┤
│ SKU TIER 3: ENTERPRISE (Free: 1,000 calls/mo | Overage: $25.00 - $32.00 / 1,000)      │
│ - places.reviews                                                                      │
│ - places.photos                                                                       │
│ - places.editorialSummary                                                             │
│ - places.paymentOptions, parkingOptions, accessibilityOptions                         │
└───────────────────────────────────────────────────────────────────────────────────────┘
```

### 💡 Optimal Field Mask for Our Service (Pro Tier)
To extract 100% of the data needed for lead generation in a **single call per 20 places** (staying within the 5,000 free monthly requests):

```http
X-Goog-FieldMask: places.id,places.displayName,places.formattedAddress,places.location,places.types,places.primaryType,places.websiteUri,places.nationalPhoneNumber,places.rating,places.userRatingCount,places.businessStatus,places.googleMapsUri,places.regularOpeningHours,nextPageToken
```

---

## 4. Error Responses & Handling

Google Places API returns standardized RFC 7807 / Google RPC error envelopes:

```json
{
  "error": {
    "code": 400,
    "message": "Field mask must be specified in the request header X-Goog-FieldMask.",
    "status": "INVALID_ARGUMENT",
    "details": [
      {
        "@type": "type.googleapis.com/google.rpc.BadRequest",
        "fieldViolations": [
          {
            "field": "X-Goog-FieldMask",
            "description": "Header missing or invalid."
          }
        ]
      }
    ]
  }
}
```

### Error Code Actions:
| HTTP Code | RPC Status | Cause | Service Action |
| :--- | :--- | :--- | :--- |
| **400** | `INVALID_ARGUMENT` | Missing field mask or bad JSON | Fix request schema / validate parameters |
| **401 / 403** | `PERMISSION_DENIED` | Invalid API key or IP restriction mismatch | Fail fast, notify user in UI |
| **429** | `RESOURCE_EXHAUSTED` | Exceeded QPS rate limit or monthly budget cap | Exponential backoff (1s, 2s, 4s) or fallback to Web Scraper (Mode A) |
| **500 / 503** | `INTERNAL` / `UNAVAILABLE` | Google backend glitch | Retry up to 3 times with jitter |
