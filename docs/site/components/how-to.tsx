import type { ReactNode } from 'react';

export interface HowToProps {
  goal: string;
  prerequisites: string[];
  steps: ReactNode[];
  verification: string | ReactNode;
  exampleFile?: string;
}

export function HowTo({
  goal,
  prerequisites,
  steps,
  verification,
  exampleFile,
}: HowToProps) {
  return (
    <div className="not-prose flex flex-col gap-6 my-6">
      <section className="rounded-lg border border-fd-border bg-fd-card p-5">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-fd-muted-foreground mb-2">
          Goal
        </h2>
        <p className="text-base text-fd-foreground">{goal}</p>
      </section>

      <section className="rounded-lg border border-fd-border bg-fd-card p-5">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-fd-muted-foreground mb-3">
          Prerequisites
        </h2>
        <ul className="list-disc list-inside space-y-1 text-fd-foreground">
          {prerequisites.map((p, i) => (
            <li key={i}>{p}</li>
          ))}
        </ul>
      </section>

      <section className="rounded-lg border border-fd-border bg-fd-card p-5">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-fd-muted-foreground mb-3">
          Steps
        </h2>
        <ol className="list-decimal list-inside space-y-2 text-fd-foreground">
          {steps.map((s, i) => (
            <li key={i} className="leading-relaxed">{s}</li>
          ))}
        </ol>
      </section>

      <section className="rounded-lg border border-fd-border bg-fd-card p-5">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-fd-muted-foreground mb-2">
          Verification
        </h2>
        <div className="text-fd-foreground">{verification}</div>
      </section>

      {exampleFile && (
        <section className="rounded-lg border border-fd-border bg-fd-card p-5">
          <h2 className="text-sm font-semibold uppercase tracking-wider text-fd-muted-foreground mb-2">
            Example file
          </h2>
          <a
            href={exampleFile}
            download
            className="inline-flex items-center gap-2 text-fd-foreground underline hover:no-underline"
          >
            Download <code className="text-sm">{exampleFile.split('/').pop()}</code>
          </a>
        </section>
      )}
    </div>
  );
}
