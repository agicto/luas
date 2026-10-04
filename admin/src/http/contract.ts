import type { operations } from '@/http/generated/openapi';

type JsonContent<Response> = Response extends { content: { 'application/json': infer Body } }
  ? Body
  : never;

/** The JSON body that `contracts/openapi.yaml` declares for one operation and status. */
export type ContractResponse<
  Operation extends keyof operations,
  Status extends keyof operations[Operation]['responses'],
> = JsonContent<operations[Operation]['responses'][Status]>;

/** The `data` member of a success envelope. */
export type ContractData<
  Operation extends keyof operations,
  Status extends keyof operations[Operation]['responses'],
> = ContractResponse<Operation, Status> extends { data: infer Data } ? Data : never;

/**
 * True when every body the contract allows is accepted by a client schema's input type. Fields
 * the client ignores are fine; a field the client requires but the contract does not guarantee,
 * or a narrower client type, fails.
 */
export type Accepts<ClientInput, Contract> = [Contract] extends [ClientInput] ? true : false;

/** Compile-time assertion: `corepack pnpm type-check` fails when a client schema drifts. */
export function assertContract<Holds extends true>(): void {
  return undefined as unknown as Holds extends true ? void : never;
}

/** The JSON request body that `contracts/openapi.yaml` declares for one operation. */
export type ContractRequest<Operation extends keyof operations> = operations[Operation] extends {
  requestBody?: infer Body;
}
  ? JsonContent<NonNullable<Body>>
  : never;

/** True when everything the client sends is a body the contract accepts. */
export type Sends<ClientBody, Contract> = [ClientBody] extends [Contract] ? true : false;
