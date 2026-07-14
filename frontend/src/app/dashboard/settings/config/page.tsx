/**
 * System configuration page — admin settings for scoring, LLM, voice, and automation.
 *
 * Server Component. Renders SystemConfigPageClient with the effective config.
 */

import type { Metadata } from "next";
import { SystemConfigPageClient } from "./SystemConfigPageClient";

export const metadata: Metadata = {
  title: "System Configuration | MyJob Agent",
  description: "Configure scoring, LLM providers, voice, approval tiers, and automation settings.",
};

export default function SystemConfigPage() {
  return <SystemConfigPageClient />;
}
