import { useState, useEffect, useRef } from "react";
import {
  Search,
  FileText,
  Send,
  Mail,
  Mic,
  Activity,
  CheckCircle,
  XCircle,
  Clock,
  Globe,
  Zap,
  ChevronRight,
  LayoutDashboard,
  Briefcase,
  ScrollText,
  Inbox,
  BrainCircuit,
  Settings,
  Play,
  Pause,
  RotateCcw,
  Eye,
  Download,
  AlertTriangle,
  TrendingUp,
  Bot,
  Terminal,
  Database,
  Cpu,
} from "lucide-react";

// ─── types ───────────────────────────────────────────────────────────────────

type Page = "dashboard" | "jobs" | "approvals" | "applications" | "emails" | "interviews" | "agent";

interface Job {
  id: string;
  title: string;
  company: string;
  location: string;
  remote: boolean;
  salary: string;
  score: number;
  tier: "auto" | "review" | "reject";
  source: string;
  postedAt: string;
  status: "new" | "queued" | "applied" | "skipped";
  tags: string[];
}

interface Approval {
  id: string;
  company: string;
  title: string;
  score: number;
  salary: string;
  remote: boolean;
  resumeVariant: string;
  coverLetterSnippet: string;
  matchedSkills: string[];
  submittedAt: string;
}

interface Application {
  id: string;
  company: string;
  title: string;
  score: number;
  appliedAt: string;
  status: "applied" | "assessment" | "phone_screen" | "technical" | "final" | "offer" | "rejected";
  tier: "auto" | "review";
}

interface LogEntry {
  id: string;
  time: string;
  type: "scrape" | "match" | "resume" | "apply" | "email" | "approve" | "reject" | "task";
  message: string;
  meta?: string;
}

// ─── mock data ───────────────────────────────────────────────────────────────

const JOBS: Job[] = [
  { id: "j1", title: "Senior Backend Engineer", company: "Vercel", location: "Remote", remote: true, salary: "$180k–$220k", score: 97, tier: "auto", source: "Greenhouse", postedAt: "2h ago", status: "applied", tags: ["Go", "Postgres", "Kubernetes"] },
  { id: "j2", title: "Staff Software Engineer", company: "PlanetScale", location: "Remote", remote: true, salary: "$200k–$240k", score: 94, tier: "review", source: "Lever", postedAt: "3h ago", status: "queued", tags: ["Go", "MySQL", "Distributed Systems"] },
  { id: "j3", title: "Platform Engineer", company: "Stripe", location: "San Francisco, CA", remote: false, salary: "$190k–$230k", score: 91, tier: "review", source: "Greenhouse", postedAt: "4h ago", status: "queued", tags: ["Go", "gRPC", "AWS"] },
  { id: "j4", title: "Backend Engineer, Infra", company: "Linear", location: "Remote", remote: true, salary: "$160k–$195k", score: 96, tier: "auto", source: "Ashby", postedAt: "5h ago", status: "applied", tags: ["TypeScript", "Postgres", "Redis"] },
  { id: "j5", title: "Software Engineer, Core", company: "Turso", location: "Remote", remote: true, salary: "$140k–$180k", score: 88, tier: "review", source: "Indeed", postedAt: "6h ago", status: "new", tags: ["Rust", "SQLite", "C"] },
  { id: "j6", title: "Senior SRE", company: "Datadog", location: "New York, NY", remote: false, salary: "$185k–$215k", score: 74, tier: "reject", source: "Workday", postedAt: "7h ago", status: "skipped", tags: ["Kubernetes", "Terraform", "Go"] },
  { id: "j7", title: "AI Infrastructure Engineer", company: "Mistral AI", location: "Paris / Remote", remote: true, salary: "€130k–€165k", score: 92, tier: "review", source: "Lever", postedAt: "8h ago", status: "new", tags: ["Python", "CUDA", "Go"] },
  { id: "j8", title: "Backend Engineer, APIs", company: "Resend", location: "Remote", remote: true, salary: "$150k–$185k", score: 98, tier: "auto", source: "Ashby", postedAt: "9h ago", status: "applied", tags: ["TypeScript", "Node.js", "Postgres"] },
];

const APPROVALS: Approval[] = [
  {
    id: "a1", company: "PlanetScale", title: "Staff Software Engineer", score: 94, salary: "$200k–$240k", remote: true,
    resumeVariant: "Backend — Distributed Systems",
    coverLetterSnippet: "My five years building high-throughput data pipelines at scale aligns directly with PlanetScale's mission to bring serverless MySQL to every team. I have shipped production systems handling 200k+ queries/sec...",
    matchedSkills: ["Go", "MySQL", "Distributed Systems", "Kubernetes", "Performance Tuning"],
    submittedAt: "12 min ago",
  },
  {
    id: "a2", company: "Stripe", title: "Platform Engineer", score: 91, salary: "$190k–$230k", remote: false,
    resumeVariant: "Backend — Cloud Infrastructure",
    coverLetterSnippet: "The Platform Engineering role at Stripe stands out because it sits at the intersection of reliability and developer experience — a space I have spent the last three years obsessing over...",
    matchedSkills: ["Go", "gRPC", "AWS", "Terraform", "SLO/SLI Design"],
    submittedAt: "34 min ago",
  },
  {
    id: "a3", company: "Mistral AI", title: "AI Infrastructure Engineer", score: 92, salary: "€130k–€165k", remote: true,
    resumeVariant: "AI/ML Engineering",
    coverLetterSnippet: "Scaling training infrastructure for frontier models requires both deep systems knowledge and an understanding of how model architecture decisions interact with hardware constraints...",
    matchedSkills: ["Python", "CUDA", "Go", "Kubernetes", "PyTorch"],
    submittedAt: "58 min ago",
  },
];

const APPLICATIONS: Application[] = [
  { id: "ap1", company: "Vercel", title: "Senior Backend Engineer", score: 97, appliedAt: "2h ago", status: "assessment", tier: "auto" },
  { id: "ap2", company: "Linear", title: "Backend Engineer, Infra", score: 96, appliedAt: "5h ago", status: "applied", tier: "auto" },
  { id: "ap3", company: "Resend", title: "Backend Engineer, APIs", score: 98, appliedAt: "9h ago", status: "phone_screen", tier: "auto" },
  { id: "ap4", company: "Neon", title: "Senior Go Engineer", score: 93, appliedAt: "1d ago", status: "technical", tier: "review" },
  { id: "ap5", company: "Fly.io", title: "Platform Engineer", score: 90, appliedAt: "2d ago", status: "final", tier: "review" },
  { id: "ap6", company: "Railway", title: "Backend Engineer", score: 88, appliedAt: "3d ago", status: "offer", tier: "review" },
  { id: "ap7", company: "Cloudflare", title: "Systems Engineer", score: 76, appliedAt: "4d ago", status: "rejected", tier: "review" },
];

const INITIAL_LOG: LogEntry[] = [
  { id: "l1", time: "14:32:01", type: "apply", message: "Auto-applied to Resend — Backend Engineer, APIs", meta: "score: 98 · Ashby form filled" },
  { id: "l2", time: "14:31:48", type: "resume", message: "Generated resume variant for Resend", meta: "Backend · 847ms · pdflatex" },
  { id: "l3", time: "14:31:03", type: "match", message: "Scored Turso — Software Engineer, Core", meta: "score: 88 → REVIEW tier" },
  { id: "l4", time: "14:30:22", type: "scrape", message: "Scraped 14 new jobs from Ashby", meta: "0.5 req/s · 2 rate-limited" },
  { id: "l5", time: "14:29:55", type: "email", message: "Recruiter email from noreply@vercel.com", meta: "classified: assessment_link" },
  { id: "l6", time: "14:29:11", type: "task", message: "Task fill_form completed", meta: "task_id: task_4f9a · 38s" },
  { id: "l7", time: "14:28:40", type: "apply", message: "Auto-applied to Linear — Backend Engineer, Infra", meta: "score: 96 · Greenhouse form filled" },
  { id: "l8", time: "14:27:53", type: "scrape", message: "Scraped 22 new jobs from Greenhouse", meta: "0.5 req/s · stealth mode" },
  { id: "l9", time: "14:26:18", type: "match", message: "Scored Datadog — Senior SRE", meta: "score: 74 → REJECT" },
  { id: "l10", time: "14:25:04", type: "resume", message: "Generated cover letter for PlanetScale", meta: "gpt-4o · 618 tokens · 1.2s" },
];

const PIPELINE_STAGES = [
  { key: "applied", label: "Applied", color: "#3b82f6" },
  { key: "assessment", label: "Assessment", color: "#fbbf24" },
  { key: "phone_screen", label: "Phone Screen", color: "#f97316" },
  { key: "technical", label: "Technical", color: "#a855f7" },
  { key: "final", label: "Final Round", color: "#00d4a0" },
  { key: "offer", label: "Offer", color: "#22c55e" },
  { key: "rejected", label: "Rejected", color: "#ff3b5b" },
];

// ─── helpers ──────────────────────────────────────────────────────────────────

function tierBadge(tier: "auto" | "review" | "reject") {
  if (tier === "auto") return <span className="px-1.5 py-0.5 text-[10px] font-mono font-medium rounded bg-emerald-500/15 text-emerald-400 border border-emerald-500/25 tracking-wider uppercase">AUTO</span>;
  if (tier === "review") return <span className="px-1.5 py-0.5 text-[10px] font-mono font-medium rounded bg-amber-500/15 text-amber-400 border border-amber-500/25 tracking-wider uppercase">REVIEW</span>;
  return <span className="px-1.5 py-0.5 text-[10px] font-mono font-medium rounded bg-red-500/15 text-red-400 border border-red-500/25 tracking-wider uppercase">SKIP</span>;
}

function scoreColor(score: number) {
  if (score >= 95) return "text-emerald-400";
  if (score >= 80) return "text-amber-400";
  return "text-red-400";
}

function logIcon(type: LogEntry["type"]) {
  const cls = "w-3 h-3 shrink-0 mt-0.5";
  if (type === "scrape") return <Globe className={`${cls} text-blue-400`} />;
  if (type === "match") return <Zap className={`${cls} text-amber-400`} />;
  if (type === "resume") return <FileText className={`${cls} text-purple-400`} />;
  if (type === "apply") return <Send className={`${cls} text-emerald-400`} />;
  if (type === "email") return <Mail className={`${cls} text-sky-400`} />;
  if (type === "approve") return <CheckCircle className={`${cls} text-emerald-400`} />;
  if (type === "reject") return <XCircle className={`${cls} text-red-400`} />;
  return <Activity className={`${cls} text-muted-foreground`} />;
}

function statusBadge(status: Application["status"]) {
  const styles: Record<string, string> = {
    applied: "bg-blue-500/15 text-blue-400 border-blue-500/25",
    assessment: "bg-amber-500/15 text-amber-400 border-amber-500/25",
    phone_screen: "bg-orange-500/15 text-orange-400 border-orange-500/25",
    technical: "bg-purple-500/15 text-purple-400 border-purple-500/25",
    final: "bg-cyan-500/15 text-cyan-400 border-cyan-500/25",
    offer: "bg-emerald-500/15 text-emerald-400 border-emerald-500/25",
    rejected: "bg-red-500/15 text-red-400 border-red-500/25",
  };
  const labels: Record<string, string> = {
    applied: "APPLIED",
    assessment: "ASSESSMENT",
    phone_screen: "PHONE",
    technical: "TECHNICAL",
    final: "FINAL",
    offer: "OFFER",
    rejected: "REJECTED",
  };
  return (
    <span className={`px-1.5 py-0.5 text-[10px] font-mono font-medium rounded border tracking-wider uppercase ${styles[status]}`}>
      {labels[status]}
    </span>
  );
}

// ─── sub-components ───────────────────────────────────────────────────────────

function StatCard({ label, value, sub, icon: Icon, accent }: { label: string; value: string | number; sub: string; icon: React.ElementType; accent?: string }) {
  return (
    <div className="bg-card border border-border rounded p-4 flex flex-col gap-3 hover:border-primary/30 transition-colors">
      <div className="flex items-center justify-between">
        <span className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase">{label}</span>
        <Icon className="w-4 h-4 text-muted-foreground" />
      </div>
      <div className={`text-3xl font-mono font-medium tracking-tight ${accent ?? "text-foreground"}`}>{value}</div>
      <div className="text-[11px] text-muted-foreground">{sub}</div>
    </div>
  );
}

function AgentStatus({ running, onToggle }: { running: boolean; onToggle: () => void }) {
  return (
    <button
      onClick={onToggle}
      className={`flex items-center gap-2 px-3 py-1.5 rounded border text-xs font-mono transition-all ${running
        ? "border-emerald-500/40 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/15"
        : "border-border bg-secondary text-muted-foreground hover:border-border/60"
        }`}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${running ? "bg-emerald-400 animate-pulse" : "bg-muted-foreground"}`} />
      {running ? "AGENT RUNNING" : "AGENT PAUSED"}
    </button>
  );
}

// ─── pages ───────────────────────────────────────────────────────────────────

function DashboardPage({ log, agentRunning }: { log: LogEntry[]; agentRunning: boolean }) {
  const logRef = useRef<HTMLDivElement>(null);

  const autoToday = JOBS.filter(j => j.tier === "auto" && j.status === "applied").length;
  const pendingApprovals = APPROVALS.length;
  const totalScraped = 247;
  const activeTasks = 4;

  return (
    <div className="flex flex-col gap-6">
      {/* Stats */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <StatCard label="Jobs Scraped" value={totalScraped} sub="Today · 12 sources active" icon={Search} accent="text-blue-400" />
        <StatCard label="Auto-Applied" value={autoToday} sub="Score ≥ 95 · no review needed" icon={Send} accent="text-emerald-400" />
        <StatCard label="Pending Review" value={pendingApprovals} sub="Score 80–94 · awaiting you" icon={Clock} accent="text-amber-400" />
        <StatCard label="Active Tasks" value={activeTasks} sub="fill_form × 2 · scrape × 2" icon={Cpu} accent="text-purple-400" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Live Activity Feed */}
        <div className="lg:col-span-2 bg-card border border-border rounded flex flex-col">
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <span className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase flex items-center gap-2">
              <Terminal className="w-3 h-3" />
              Live Activity
            </span>
            <span className={`text-[10px] font-mono ${agentRunning ? "text-emerald-400" : "text-muted-foreground"}`}>
              {agentRunning ? "● STREAMING" : "○ PAUSED"}
            </span>
          </div>
          <div ref={logRef} className="flex-1 overflow-y-auto max-h-80 p-3 space-y-1 scrollbar-hide">
            {log.map(entry => (
              <div key={entry.id} className="flex items-start gap-2 py-1 hover:bg-secondary/40 rounded px-1 group">
                {logIcon(entry.type)}
                <div className="flex-1 min-w-0">
                  <div className="flex items-baseline gap-2">
                    <span className="text-[10px] font-mono text-muted-foreground shrink-0">{entry.time}</span>
                    <span className="text-xs text-foreground truncate">{entry.message}</span>
                  </div>
                  {entry.meta && <div className="text-[10px] font-mono text-muted-foreground mt-0.5 truncate">{entry.meta}</div>}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Pipeline Summary */}
        <div className="bg-card border border-border rounded flex flex-col">
          <div className="px-4 py-3 border-b border-border">
            <span className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase flex items-center gap-2">
              <TrendingUp className="w-3 h-3" />
              Pipeline
            </span>
          </div>
          <div className="flex-1 p-4 space-y-2">
            {PIPELINE_STAGES.map(stage => {
              const count = APPLICATIONS.filter(a => a.status === stage.key).length;
              return (
                <div key={stage.key} className="flex items-center gap-3">
                  <span className="text-[10px] font-mono text-muted-foreground w-20 shrink-0">{stage.label}</span>
                  <div className="flex-1 bg-secondary rounded-full h-1.5 overflow-hidden">
                    <div
                      className="h-full rounded-full transition-all"
                      style={{ width: count > 0 ? `${Math.max(count * 22, 8)}%` : "0%", backgroundColor: stage.color }}
                    />
                  </div>
                  <span className="text-[10px] font-mono text-muted-foreground w-4 text-right">{count}</span>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* Browser Agent Preview */}
      <div className="bg-card border border-border rounded">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <span className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase flex items-center gap-2">
            <Bot className="w-3 h-3" />
            Browser Agent · Current Task
          </span>
          <span className="text-[10px] font-mono text-amber-400 bg-amber-500/10 border border-amber-500/25 px-2 py-0.5 rounded">fill_form · 62% complete</span>
        </div>
        <div className="p-4">
          <div className="bg-[#060a0d] border border-border/50 rounded overflow-hidden">
            <div className="flex items-center gap-1.5 px-3 py-2 border-b border-border/50 bg-secondary/30">
              <div className="w-2 h-2 rounded-full bg-red-500/60" />
              <div className="w-2 h-2 rounded-full bg-amber-500/60" />
              <div className="w-2 h-2 rounded-full bg-emerald-500/60" />
              <div className="flex-1 mx-3 bg-secondary rounded px-2 py-0.5 text-[10px] font-mono text-muted-foreground truncate">
                https://boards.greenhouse.io/stripe/jobs/5821094 — Platform Engineer
              </div>
            </div>
            <div className="p-4 space-y-3">
              <div className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase">Filling Application Form — Stripe</div>
              <div className="space-y-2">
                {[
                  { label: "First Name", value: "Alex", done: true },
                  { label: "Last Name", value: "Chen", done: true },
                  { label: "Email", value: "alex@example.com", done: true },
                  { label: "Resume Upload", value: "backend_stripe_2026.pdf", done: true },
                  { label: "Cover Letter", value: "Uploading…", done: false, active: true },
                  { label: "Work Authorization", value: "—", done: false },
                  { label: "LinkedIn URL", value: "—", done: false },
                ].map((field, i) => (
                  <div key={i} className={`flex items-center gap-3 p-2 rounded text-xs ${field.active ? "bg-primary/5 border border-primary/20" : "border border-transparent"}`}>
                    <div className={`w-3 h-3 rounded-full shrink-0 flex items-center justify-center ${field.done ? "bg-emerald-500/20 border border-emerald-500/40" : field.active ? "border border-primary/60 bg-primary/10" : "border border-border"}`}>
                      {field.done && <div className="w-1.5 h-1.5 rounded-full bg-emerald-400" />}
                      {field.active && <div className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />}
                    </div>
                    <span className="font-mono text-[10px] text-muted-foreground w-28 shrink-0">{field.label}</span>
                    <span className={`font-mono text-[10px] ${field.done ? "text-foreground" : field.active ? "text-primary" : "text-muted-foreground/40"}`}>{field.value}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function JobsPage() {
  const [filter, setFilter] = useState<"all" | "auto" | "review" | "reject">("all");

  const filtered = filter === "all" ? JOBS : JOBS.filter(j => j.tier === filter);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2 flex-wrap">
        {(["all", "auto", "review", "reject"] as const).map(f => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-3 py-1.5 text-[11px] font-mono uppercase tracking-wider rounded border transition-all ${filter === f
              ? "bg-primary text-primary-foreground border-primary"
              : "bg-secondary border-border text-muted-foreground hover:border-primary/40 hover:text-foreground"
              }`}
          >
            {f === "all" ? `All (${JOBS.length})` : f === "auto" ? `Auto (${JOBS.filter(j => j.tier === "auto").length})` : f === "review" ? `Review (${JOBS.filter(j => j.tier === "review").length})` : `Skip (${JOBS.filter(j => j.tier === "reject").length})`}
          </button>
        ))}
        <div className="ml-auto text-[11px] font-mono text-muted-foreground">
          Scraped today: <span className="text-foreground">247</span>
        </div>
      </div>

      <div className="space-y-2">
        {filtered.map(job => (
          <div key={job.id} className={`bg-card border rounded p-4 flex items-start gap-4 hover:border-primary/30 transition-all group ${job.tier === "reject" ? "opacity-50 border-border" : "border-border"}`}>
            <div className="w-10 h-10 rounded bg-secondary flex items-center justify-center shrink-0 border border-border">
              <span className="text-sm font-mono font-medium text-foreground">{job.company[0]}</span>
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-start justify-between gap-2 flex-wrap">
                <div>
                  <div className="text-sm font-medium text-foreground">{job.title}</div>
                  <div className="text-xs text-muted-foreground mt-0.5">{job.company} · {job.location} {job.remote && <span className="text-emerald-400">· Remote</span>}</div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <span className={`text-xl font-mono font-medium ${scoreColor(job.score)}`}>{job.score}</span>
                  {tierBadge(job.tier)}
                </div>
              </div>
              <div className="flex items-center gap-3 mt-2 flex-wrap">
                <span className="text-[11px] font-mono text-muted-foreground">{job.salary}</span>
                <span className="text-[11px] font-mono text-muted-foreground">via {job.source}</span>
                <span className="text-[11px] font-mono text-muted-foreground">{job.postedAt}</span>
                <div className="flex gap-1 flex-wrap">
                  {job.tags.map(tag => (
                    <span key={tag} className="px-1.5 py-0.5 text-[10px] font-mono bg-secondary border border-border rounded text-muted-foreground">{tag}</span>
                  ))}
                </div>
              </div>
            </div>
            <div className="flex items-center gap-1.5 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
              <button className="p-1.5 rounded border border-border hover:border-primary/40 text-muted-foreground hover:text-foreground transition-colors">
                <Eye className="w-3.5 h-3.5" />
              </button>
              <button className="p-1.5 rounded border border-border hover:border-primary/40 text-muted-foreground hover:text-foreground transition-colors">
                <Send className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function ApprovalsPage() {
  const [dismissed, setDismissed] = useState<string[]>([]);
  const active = APPROVALS.filter(a => !dismissed.includes(a.id));

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase">
          {active.length} applications awaiting your review
        </div>
        <div className="flex items-center gap-1.5">
          <AlertTriangle className="w-3.5 h-3.5 text-amber-400" />
          <span className="text-[11px] font-mono text-amber-400">Score 80–94 · manual approval required</span>
        </div>
      </div>

      {active.length === 0 && (
        <div className="bg-card border border-border rounded p-12 text-center">
          <CheckCircle className="w-8 h-8 text-emerald-400 mx-auto mb-3" />
          <div className="text-sm font-mono text-muted-foreground">All caught up. No pending approvals.</div>
        </div>
      )}

      {active.map(approval => (
        <div key={approval.id} className="bg-card border border-border rounded overflow-hidden hover:border-primary/20 transition-colors">
          <div className="flex items-center justify-between p-4 border-b border-border">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded bg-secondary border border-border flex items-center justify-center">
                <span className="text-sm font-mono font-medium">{approval.company[0]}</span>
              </div>
              <div>
                <div className="text-sm font-medium">{approval.title}</div>
                <div className="text-xs text-muted-foreground">{approval.company} · {approval.salary} {approval.remote && <span className="text-emerald-400">· Remote</span>}</div>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <div className="text-right">
                <div className={`text-2xl font-mono font-medium ${scoreColor(approval.score)}`}>{approval.score}</div>
                <div className="text-[10px] font-mono text-muted-foreground">match score</div>
              </div>
              {tierBadge("review")}
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-0 divide-y md:divide-y-0 md:divide-x divide-border">
            <div className="p-4">
              <div className="text-[10px] font-mono text-muted-foreground tracking-widest uppercase mb-2">Resume Variant</div>
              <div className="text-xs font-mono text-primary mb-3">{approval.resumeVariant}</div>
              <div className="text-[10px] font-mono text-muted-foreground tracking-widest uppercase mb-2">Matched Skills</div>
              <div className="flex flex-wrap gap-1">
                {approval.matchedSkills.map(s => (
                  <span key={s} className="px-1.5 py-0.5 text-[10px] font-mono bg-primary/10 border border-primary/25 rounded text-primary">{s}</span>
                ))}
              </div>
            </div>
            <div className="p-4">
              <div className="text-[10px] font-mono text-muted-foreground tracking-widest uppercase mb-2">Cover Letter Preview</div>
              <p className="text-xs text-foreground/70 leading-relaxed italic">&ldquo;{approval.coverLetterSnippet}&rdquo;</p>
              <div className="flex items-center gap-1 mt-2">
                <button className="text-[10px] font-mono text-muted-foreground hover:text-foreground flex items-center gap-1 transition-colors">
                  <Eye className="w-3 h-3" /> View full
                </button>
                <span className="text-muted-foreground/30 mx-1">·</span>
                <button className="text-[10px] font-mono text-muted-foreground hover:text-foreground flex items-center gap-1 transition-colors">
                  <Download className="w-3 h-3" /> Resume PDF
                </button>
              </div>
            </div>
          </div>

          <div className="flex items-center justify-between p-3 border-t border-border bg-secondary/20">
            <span className="text-[10px] font-mono text-muted-foreground">Queued {approval.submittedAt}</span>
            <div className="flex gap-2">
              <button
                onClick={() => setDismissed(d => [...d, approval.id])}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono border border-red-500/30 bg-red-500/10 text-red-400 rounded hover:bg-red-500/20 transition-colors"
              >
                <XCircle className="w-3.5 h-3.5" />
                Reject
              </button>
              <button
                onClick={() => setDismissed(d => [...d, approval.id])}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono border border-primary/40 bg-primary/10 text-primary rounded hover:bg-primary/20 transition-colors"
              >
                <CheckCircle className="w-3.5 h-3.5" />
                Approve & Submit
              </button>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function ApplicationsPage() {
  return (
    <div className="flex flex-col gap-4">
      <div className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase">
        {APPLICATIONS.length} total applications · {APPLICATIONS.filter(a => a.status !== "rejected").length} active
      </div>

      {/* Kanban-style pipeline */}
      <div className="overflow-x-auto pb-2">
        <div className="flex gap-3 min-w-max">
          {PIPELINE_STAGES.map(stage => {
            const apps = APPLICATIONS.filter(a => a.status === stage.key);
            return (
              <div key={stage.key} className="w-52 shrink-0">
                <div className="flex items-center gap-2 mb-2 px-1">
                  <div className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: stage.color }} />
                  <span className="text-[10px] font-mono text-muted-foreground tracking-widest uppercase">{stage.label}</span>
                  <span className="text-[10px] font-mono text-muted-foreground ml-auto">{apps.length}</span>
                </div>
                <div className="space-y-2">
                  {apps.map(app => (
                    <div key={app.id} className="bg-card border border-border rounded p-3 hover:border-primary/30 transition-colors cursor-default">
                      <div className="flex items-start justify-between gap-1 mb-1.5">
                        <div className="text-xs font-medium leading-tight">{app.company}</div>
                        <span className={`text-xs font-mono font-medium shrink-0 ${scoreColor(app.score)}`}>{app.score}</span>
                      </div>
                      <div className="text-[10px] text-muted-foreground leading-tight mb-2">{app.title}</div>
                      <div className="flex items-center justify-between">
                        {tierBadge(app.tier)}
                        <span className="text-[9px] font-mono text-muted-foreground">{app.appliedAt}</span>
                      </div>
                    </div>
                  ))}
                  {apps.length === 0 && (
                    <div className="bg-card/40 border border-dashed border-border/50 rounded p-3 text-center">
                      <span className="text-[10px] font-mono text-muted-foreground/40">empty</span>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Table view */}
      <div className="bg-card border border-border rounded overflow-hidden">
        <div className="px-4 py-3 border-b border-border">
          <span className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase">All Applications</span>
        </div>
        <div className="divide-y divide-border">
          {APPLICATIONS.map(app => (
            <div key={app.id} className="flex items-center gap-4 px-4 py-3 hover:bg-secondary/30 transition-colors">
              <div className="w-7 h-7 rounded bg-secondary border border-border flex items-center justify-center shrink-0">
                <span className="text-[10px] font-mono font-medium">{app.company[0]}</span>
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-xs font-medium">{app.title}</div>
                <div className="text-[10px] text-muted-foreground">{app.company}</div>
              </div>
              <span className={`text-sm font-mono font-medium ${scoreColor(app.score)}`}>{app.score}</span>
              {statusBadge(app.status)}
              {tierBadge(app.tier)}
              <span className="text-[10px] font-mono text-muted-foreground shrink-0 hidden sm:block">{app.appliedAt}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function AgentPage() {
  const tasks = [
    { id: "task_4f9a", type: "fill_form", status: "active", progress: 62, company: "Stripe", detail: "Filling Greenhouse form · Cover Letter upload", started: "4m ago" },
    { id: "task_3b8c", type: "scrape_source", status: "active", progress: 38, company: "Lever", detail: "Scraping 47 of 120 job listings", started: "2m ago" },
    { id: "task_9e1d", type: "generate_resume", status: "pending", progress: 0, company: "Mistral AI", detail: "Waiting for task_4f9a", started: "—" },
    { id: "task_7a2f", type: "generate_coverletter", status: "pending", progress: 0, company: "Turso", detail: "Queued · gpt-4o", started: "—" },
    { id: "task_2c6b", type: "fill_form", status: "completed", progress: 100, company: "Linear", detail: "Ashby form submitted successfully", started: "12m ago" },
    { id: "task_5d3a", type: "sync_emails", status: "completed", progress: 100, company: "—", detail: "8 new emails synced · 2 classified", started: "18m ago" },
  ];

  const statusColors: Record<string, string> = {
    active: "text-emerald-400",
    pending: "text-amber-400",
    completed: "text-muted-foreground",
    failed: "text-red-400",
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {[
          { label: "Active Tasks", value: 2, color: "text-emerald-400" },
          { label: "Queued", value: 2, color: "text-amber-400" },
          { label: "Completed Today", value: 89, color: "text-muted-foreground" },
          { label: "Failed", value: 1, color: "text-red-400" },
        ].map(stat => (
          <div key={stat.label} className="bg-card border border-border rounded p-3">
            <div className="text-[10px] font-mono text-muted-foreground tracking-widest uppercase mb-1">{stat.label}</div>
            <div className={`text-2xl font-mono font-medium ${stat.color}`}>{stat.value}</div>
          </div>
        ))}
      </div>

      <div className="bg-card border border-border rounded overflow-hidden">
        <div className="px-4 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase flex items-center gap-2">
            <Database className="w-3 h-3" />
            Task Queue
          </span>
          <button className="flex items-center gap-1.5 text-[10px] font-mono text-muted-foreground hover:text-foreground transition-colors border border-border rounded px-2 py-1">
            <RotateCcw className="w-3 h-3" />
            Refresh
          </button>
        </div>
        <div className="divide-y divide-border">
          {tasks.map(task => (
            <div key={task.id} className="px-4 py-3 hover:bg-secondary/20 transition-colors">
              <div className="flex items-start justify-between gap-3">
                <div className="flex items-start gap-3 min-w-0">
                  <div className={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${task.status === "active" ? "bg-emerald-400 animate-pulse" : task.status === "pending" ? "bg-amber-400" : task.status === "completed" ? "bg-muted-foreground/40" : "bg-red-400"}`} />
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-[10px] font-mono bg-secondary border border-border rounded px-1.5 py-0.5 text-muted-foreground">{task.type}</span>
                      <span className="text-xs font-medium">{task.company !== "—" ? task.company : ""}</span>
                      <span className={`text-[10px] font-mono uppercase ${statusColors[task.status]}`}>{task.status}</span>
                    </div>
                    <div className="text-[11px] text-muted-foreground mt-1">{task.detail}</div>
                  </div>
                </div>
                <div className="text-right shrink-0">
                  <div className="text-[10px] font-mono text-muted-foreground">{task.id}</div>
                  <div className="text-[10px] font-mono text-muted-foreground">{task.started}</div>
                </div>
              </div>
              {task.status === "active" && (
                <div className="mt-2 ml-5">
                  <div className="flex items-center gap-2">
                    <div className="flex-1 bg-secondary rounded-full h-1 overflow-hidden">
                      <div className="h-full bg-primary rounded-full transition-all" style={{ width: `${task.progress}%` }} />
                    </div>
                    <span className="text-[10px] font-mono text-primary">{task.progress}%</span>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* System health */}
      <div className="bg-card border border-border rounded">
        <div className="px-4 py-3 border-b border-border">
          <span className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase">System Health</span>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-0 divide-x divide-border">
          {[
            { service: "Go API", status: "online", port: ":8080", latency: "2ms" },
            { service: "Worker", status: "online", port: "asynq", latency: "—" },
            { service: "Browser Agent", status: "online", port: ":3000", latency: "12ms" },
            { service: "Ollama", status: "online", port: ":11434", latency: "230ms" },
          ].map(svc => (
            <div key={svc.service} className="p-4">
              <div className="flex items-center gap-1.5 mb-1">
                <div className={`w-1.5 h-1.5 rounded-full ${svc.status === "online" ? "bg-emerald-400" : "bg-red-400"}`} />
                <span className="text-[10px] font-mono text-muted-foreground uppercase tracking-wider">{svc.service}</span>
              </div>
              <div className="text-xs font-mono text-foreground">{svc.port}</div>
              {svc.latency !== "—" && <div className="text-[10px] font-mono text-muted-foreground">{svc.latency}</div>}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ─── main app ─────────────────────────────────────────────────────────────────

const NAV = [
  { key: "dashboard", label: "Dashboard", icon: LayoutDashboard },
  { key: "jobs", label: "Jobs Queue", icon: Briefcase },
  { key: "approvals", label: "Approvals", icon: ScrollText, badge: 3 },
  { key: "applications", label: "Applications", icon: Send },
  { key: "emails", label: "Emails", icon: Inbox, badge: 2 },
  { key: "interviews", label: "Interviews", icon: Mic },
  { key: "agent", label: "Agent Tasks", icon: BrainCircuit },
] as const;

export default function App() {
  const [page, setPage] = useState<Page>("dashboard");
  const [agentRunning, setAgentRunning] = useState(true);
  const [log, setLog] = useState<LogEntry[]>(INITIAL_LOG);
  const [sidebarOpen, setSidebarOpen] = useState(true);

  const NEW_ENTRIES: LogEntry[] = [
    { id: "", time: "", type: "scrape", message: "Scraped 8 new jobs from Indeed", meta: "0.5 req/s · stealth mode active" },
    { id: "", time: "", type: "match", message: "Scored Fly.io — Backend Engineer Rust", meta: "score: 86 → REVIEW tier" },
    { id: "", time: "", type: "resume", message: "Generated resume variant for Turso", meta: "ai specialization · 1.1s" },
    { id: "", time: "", type: "apply", message: "Auto-applied to Neon — Staff Engineer", meta: "score: 96 · Ashby form filled" },
    { id: "", time: "", type: "task", message: "Task scrape_source completed", meta: "task_id: task_8c3d · 47 jobs · 2m14s" },
    { id: "", time: "", type: "email", message: "Draft reply generated for Fly.io recruiter", meta: "gpt-4o · warm interest tone" },
  ];

  useEffect(() => {
    if (!agentRunning) return;
    const interval = setInterval(() => {
      const template = NEW_ENTRIES[Math.floor(Math.random() * NEW_ENTRIES.length)];
      const now = new Date();
      const newEntry: LogEntry = {
        ...template,
        id: `l${Date.now()}`,
        time: `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}:${String(now.getSeconds()).padStart(2, "0")}`,
      };
      setLog(prev => [newEntry, ...prev.slice(0, 49)]);
    }, 3500);
    return () => clearInterval(interval);
  }, [agentRunning]);

  const pageTitle: Record<Page, string> = {
    dashboard: "Dashboard",
    jobs: "Jobs Queue",
    approvals: "Pending Approvals",
    applications: "Applications",
    emails: "Email Inbox",
    interviews: "Interview Prep",
    agent: "Agent Tasks",
  };

  return (
    <div className="flex h-screen bg-background text-foreground overflow-hidden" style={{ fontFamily: "'Inter Tight', system-ui, sans-serif" }}>
      {/* Sidebar */}
      <aside
        className={`shrink-0 bg-sidebar border-r border-sidebar-border flex-col hidden sm:flex transition-all duration-200 ${sidebarOpen ? "w-56" : "w-14"}`}
      >
        {/* Logo / collapse toggle */}
        <div className={`border-b border-sidebar-border flex items-center ${sidebarOpen ? "px-4 py-5 justify-between" : "px-0 py-4 justify-center"}`}>
          {sidebarOpen && (
            <div className="flex items-center gap-2.5 min-w-0">
              <div className="w-7 h-7 rounded bg-primary flex items-center justify-center shrink-0">
                <Bot className="w-4 h-4 text-primary-foreground" />
              </div>
              <div className="min-w-0">
                <div className="text-sm font-mono font-medium text-foreground leading-none">MyJob</div>
                <div className="text-[10px] font-mono text-muted-foreground mt-0.5">AI Agent v2.0</div>
              </div>
            </div>
          )}
          {!sidebarOpen && (
            <div className="w-7 h-7 rounded bg-primary flex items-center justify-center shrink-0">
              <Bot className="w-4 h-4 text-primary-foreground" />
            </div>
          )}
          <button
            onClick={() => setSidebarOpen(o => !o)}
            className={`p-1 rounded text-muted-foreground hover:text-foreground hover:bg-sidebar-accent/60 transition-colors ${!sidebarOpen ? "absolute left-3 top-[52px]" : ""}`}
            title={sidebarOpen ? "Collapse sidebar" : "Expand sidebar"}
          >
            <ChevronRight className={`w-3.5 h-3.5 transition-transform duration-200 ${sidebarOpen ? "rotate-180" : ""}`} />
          </button>
        </div>

        {/* Nav */}
        <nav className={`flex-1 overflow-y-auto space-y-0.5 scrollbar-hide ${sidebarOpen ? "p-2" : "p-1.5 pt-2"}`}>
          {NAV.map(item => {
            const Icon = item.icon;
            const active = page === item.key;
            return (
              <button
                key={item.key}
                onClick={() => setPage(item.key as Page)}
                title={!sidebarOpen ? item.label : undefined}
                className={`w-full flex items-center rounded text-left transition-all group relative ${sidebarOpen ? "gap-2.5 px-3 py-2" : "justify-center px-0 py-2.5"} ${active
                  ? "bg-sidebar-accent text-sidebar-accent-foreground border border-sidebar-border"
                  : "text-muted-foreground hover:text-sidebar-foreground hover:bg-sidebar-accent/50 border border-transparent"
                  }`}
              >
                <Icon className={`w-3.5 h-3.5 shrink-0 ${active ? "text-primary" : "text-muted-foreground group-hover:text-sidebar-foreground"}`} />
                {sidebarOpen && (
                  <>
                    <span className="text-xs font-medium flex-1">{item.label}</span>
                    {"badge" in item && item.badge > 0 && (
                      <span className="text-[9px] font-mono bg-amber-500/20 text-amber-400 border border-amber-500/30 rounded-full w-4 h-4 flex items-center justify-center leading-none shrink-0">
                        {item.badge}
                      </span>
                    )}
                    {active && <ChevronRight className="w-3 h-3 text-primary shrink-0" />}
                  </>
                )}
                {!sidebarOpen && "badge" in item && item.badge > 0 && (
                  <span className="absolute top-1 right-1 w-3.5 h-3.5 text-[8px] font-mono bg-amber-500/80 text-black rounded-full flex items-center justify-center leading-none">
                    {item.badge}
                  </span>
                )}
              </button>
            );
          })}
        </nav>

        {/* Bottom */}
        <div className={`border-t border-sidebar-border space-y-1 ${sidebarOpen ? "p-3" : "p-1.5"}`}>
          {sidebarOpen ? (
            <>
              <AgentStatus running={agentRunning} onToggle={() => setAgentRunning(r => !r)} />
              <button className="w-full flex items-center gap-2.5 px-3 py-2 rounded text-muted-foreground hover:text-foreground hover:bg-sidebar-accent/50 transition-colors">
                <Settings className="w-3.5 h-3.5 shrink-0" />
                <span className="text-xs font-medium">Settings</span>
              </button>
            </>
          ) : (
            <>
              <button
                onClick={() => setAgentRunning(r => !r)}
                title={agentRunning ? "Pause agent" : "Start agent"}
                className="w-full flex justify-center py-2 rounded border border-transparent hover:bg-sidebar-accent/50 transition-colors"
              >
                <span className={`w-2 h-2 rounded-full ${agentRunning ? "bg-emerald-400 animate-pulse" : "bg-muted-foreground"}`} />
              </button>
              <button title="Settings" className="w-full flex justify-center py-2 rounded text-muted-foreground hover:text-foreground hover:bg-sidebar-accent/50 transition-colors">
                <Settings className="w-3.5 h-3.5" />
              </button>
            </>
          )}
        </div>
      </aside>

      {/* Main */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Topbar */}
        <header className="h-12 border-b border-border flex items-center justify-between px-4 shrink-0">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setSidebarOpen(o => !o)}
              className="hidden sm:flex p-1.5 rounded text-muted-foreground hover:text-foreground hover:bg-secondary/60 transition-colors"
              title="Toggle sidebar"
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
                <rect x="1" y="1" width="12" height="12" rx="1.5" />
                <line x1="5" y1="1" x2="5" y2="13" />
              </svg>
            </button>
            <Bot className="w-4 h-4 text-primary sm:hidden" />
            <h1 className="text-sm font-mono font-medium text-foreground">{pageTitle[page]}</h1>
          </div>
          <div className="flex items-center gap-3">
            <div className="hidden md:flex items-center gap-2 text-[10px] font-mono text-muted-foreground">
              <span className="text-emerald-400 flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse inline-block" />
                247 scraped
              </span>
              <span className="text-border">·</span>
              <span>3 applied</span>
              <span className="text-border">·</span>
              <span className="text-amber-400">3 pending</span>
            </div>
            <div className="sm:hidden">
              <AgentStatus running={agentRunning} onToggle={() => setAgentRunning(r => !r)} />
            </div>
          </div>
        </header>

        {/* Content */}
        <main className="flex-1 overflow-y-auto p-5 scrollbar-hide">
          {page === "dashboard" && <DashboardPage log={log} agentRunning={agentRunning} />}
          {page === "jobs" && <JobsPage />}
          {page === "approvals" && <ApprovalsPage />}
          {page === "applications" && <ApplicationsPage />}
          {page === "agent" && <AgentPage />}
          {page === "emails" && (
            <div className="flex flex-col gap-4">
              <div className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase">8 threads · 2 unread</div>
              {[
                { from: "talent@vercel.com", subject: "Next steps: Senior Backend Engineer", preview: "Hi Alex, thanks for your application — our hiring team would like to invite you to complete a short technical assessment...", time: "1h ago", unread: true, tag: "assessment_link" },
                { from: "recruiting@fly.io", subject: "Your application to Fly.io", preview: "We've reviewed your application and would love to schedule a quick intro call to learn more about your background...", time: "3h ago", unread: true, tag: "recruiter_outreach" },
                { from: "no-reply@greenhouse.io", subject: "Application received — Linear", preview: "We've received your application for Backend Engineer, Infra at Linear. Our team will be in touch shortly...", time: "6h ago", unread: false, tag: "confirmation" },
                { from: "jobs@neon.tech", subject: "Interview invitation — Staff Engineer", preview: "Congratulations on advancing to the technical interview stage! Please use the link below to schedule your slot...", time: "1d ago", unread: false, tag: "interview_invite" },
              ].map((email, i) => (
                <div key={i} className={`bg-card border rounded p-4 hover:border-primary/30 transition-colors ${email.unread ? "border-primary/20" : "border-border"}`}>
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-xs font-mono text-muted-foreground">{email.from}</span>
                        {email.unread && <span className="w-1.5 h-1.5 rounded-full bg-primary shrink-0" />}
                      </div>
                      <div className={`text-sm mb-1 ${email.unread ? "font-medium text-foreground" : "text-foreground/80"}`}>{email.subject}</div>
                      <p className="text-xs text-muted-foreground leading-relaxed line-clamp-2">{email.preview}</p>
                    </div>
                    <div className="text-right shrink-0">
                      <div className="text-[10px] font-mono text-muted-foreground mb-2">{email.time}</div>
                      <span className={`text-[9px] font-mono uppercase tracking-wider px-1.5 py-0.5 rounded border ${email.tag === "assessment_link" || email.tag === "interview_invite" ? "bg-amber-500/10 text-amber-400 border-amber-500/25" : email.tag === "recruiter_outreach" ? "bg-blue-500/10 text-blue-400 border-blue-500/25" : "bg-secondary text-muted-foreground border-border"}`}>
                        {email.tag.replace(/_/g, " ")}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
          {page === "interviews" && (
            <div className="flex flex-col gap-4">
              <div className="text-[11px] font-mono text-muted-foreground tracking-widest uppercase">2 upcoming interviews</div>
              {[
                { company: "Neon", role: "Staff Engineer", type: "Technical Interview", date: "Jun 18 · 2:00 PM", prep: 87, topics: ["Distributed Systems", "PostgreSQL Internals", "Go Concurrency", "System Design"], questions: 48 },
                { company: "Fly.io", role: "Platform Engineer", type: "Final Round", date: "Jun 20 · 11:00 AM", prep: 62, topics: ["Kubernetes", "Firecracker VMs", "Networking", "WASM"], questions: 35 },
              ].map((interview, i) => (
                <div key={i} className="bg-card border border-border rounded overflow-hidden hover:border-primary/20 transition-colors">
                  <div className="flex items-center justify-between p-4 border-b border-border">
                    <div className="flex items-center gap-3">
                      <div className="w-9 h-9 rounded bg-secondary border border-border flex items-center justify-center">
                        <span className="text-sm font-mono font-medium">{interview.company[0]}</span>
                      </div>
                      <div>
                        <div className="text-sm font-medium">{interview.role}</div>
                        <div className="text-xs text-muted-foreground">{interview.company} · {interview.type}</div>
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="text-xs font-mono text-foreground">{interview.date}</div>
                      <div className="text-[10px] font-mono text-muted-foreground mt-0.5">{interview.questions} practice questions</div>
                    </div>
                  </div>
                  <div className="p-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-[10px] font-mono text-muted-foreground uppercase tracking-widest">Prep Progress</span>
                      <span className={`text-[10px] font-mono font-medium ${interview.prep >= 80 ? "text-emerald-400" : "text-amber-400"}`}>{interview.prep}%</span>
                    </div>
                    <div className="bg-secondary rounded-full h-1.5 mb-4 overflow-hidden">
                      <div className="h-full bg-primary rounded-full" style={{ width: `${interview.prep}%` }} />
                    </div>
                    <div className="flex flex-wrap gap-1 mb-3">
                      {interview.topics.map(t => (
                        <span key={t} className="px-1.5 py-0.5 text-[10px] font-mono bg-secondary border border-border rounded text-muted-foreground">{t}</span>
                      ))}
                    </div>
                    <div className="flex gap-2">
                      <button className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono border border-primary/40 bg-primary/10 text-primary rounded hover:bg-primary/20 transition-colors">
                        <Eye className="w-3 h-3" />
                        Study Plan
                      </button>
                      <button className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono border border-border bg-secondary text-muted-foreground rounded hover:border-primary/30 hover:text-foreground transition-colors">
                        <Mic className="w-3 h-3" />
                        Mock Interview
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
