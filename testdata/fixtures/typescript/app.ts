/**
 * Summarizes a support ticket with Claude.
 *
 * @expected-load
 *   monthly_calls: 100_000
 *   avg_input_tokens: 1_200
 *   avg_output_tokens: 300
 *   confidence: medium
 */
export async function summarizeTicket(body: string): Promise<string> {
  return body;
}
