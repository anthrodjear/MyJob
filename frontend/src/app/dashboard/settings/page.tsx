/**
 * Settings page — profile management with preferences, skills, education, and links.
 *
 * Server Component. Renders SettingsPageClient with the profile data.
 */

import type { Metadata } from "next";
import { SettingsPageClient } from "./SettingsPageClient";

export const metadata: Metadata = {
  title: "Settings | MyJob Agent",
  description: "Manage your profile, skills, education, and application preferences.",
};

export default function SettingsPage() {
  return <SettingsPageClient />;
}
