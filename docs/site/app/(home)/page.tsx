import Link from 'next/link';

export default function HomePage() {
  return (
    <main className="flex flex-1 flex-col justify-center text-center px-4 py-16">
      <h1 className="mb-4 text-4xl font-bold">go-udap</h1>
      <p className="text-fd-muted-foreground mb-8">
        Squeezebox UDAP configuration tool — documentation site under construction.
      </p>
      <p className="text-fd-muted-foreground">
        <Link className="text-fd-foreground underline" href="/docs">
          Browse the docs
        </Link>
      </p>
    </main>
  );
}
