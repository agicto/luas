# Capabilities

> **Technical capability layer**

`capabilities` provides reusable technical processing that is independent of business workflows.
Modules, infrastructure, and assembly code may call these capabilities, but a capability does not
own HTTP routes, product flows, or persistent workflows.

## Architecture Position

```text
internal/modules -------+
                        v
internal/infra --> internal/capabilities --> pkg
```

Import direction follows [`../../docs/PACKAGE_BOUNDARIES.md`](../../docs/PACKAGE_BOUNDARIES.md):
capabilities may depend on `pkg/` and the standard library, but must not depend on
`internal/infra/` or `internal/modules/`.

## Available Capabilities

| Capability | Package | Description |
| --- | --- | --- |
| **idgen** | `capabilities/idgen` | UUID, Snowflake, and NanoID generation |
| **crypto** | `capabilities/crypto` | Encryption, decryption, hashing, and password utilities |
| **ai** | `capabilities/ai` | Provider-neutral, bounded AI execution with an OpenAI Responses API adapter |

## Example

```go
import (
    "github.com/zgiai/luas/api/internal/capabilities/crypto"
    "github.com/zgiai/luas/api/internal/capabilities/idgen"
)

// ID generation
uuid := idgen.UUID()             // UUID v4
snowflake := idgen.Snowflake()   // Snowflake ID
nanoID := idgen.NanoID()         // NanoID
shortID := idgen.ShortID()       // Short ID

// Encryption and decryption
enc := crypto.NewAESEncryptorFromString("secret-key")
ciphertext, _ := enc.EncryptString("sensitive data")
plaintext, _ := enc.DecryptString(ciphertext)

// Password hashing
hash, _ := crypto.HashPassword("password")
ok := crypto.VerifyPassword("password", hash)

// HMAC signing
signature := crypto.HMACSHA256Hex("data", "key")
```

## Design Rules

### Do

- Provide one explicit technical capability.
- Name interfaces with a verb and object, such as `Encrypt()` or `Generate()`.
- Hide implementation details from callers.
- Depend only on the standard library, `pkg/`, or subpackages of the same capability.

### Do Not

- Add business decisions or workflow rules.
- Own HTTP routes, product flows, or starter registration.
- Depend on `internal/infra/` or `internal/modules/`.
- Become an undifferentiated `utils`, `helpers`, or `common` package.

## Adding A Capability

1. Create a focused directory under `capabilities/`.
2. Define the interface and implementation.
3. Add focused tests.
4. Update this README.
