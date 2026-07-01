// ComingSoon — placeholder page for routes not yet implemented.
// Shows a friendly card with the page icon and a message.

export default function ComingSoon({ icon = "construction", title = "Coming Soon", description }) {
  return (
    <div className="flex flex-col items-center justify-center py-20 gap-6">
      <div className="flex items-center justify-center size-20 rounded-2xl bg-brand-500/10 text-brand-500">
        <span className="material-symbols-outlined text-[40px]">{icon}</span>
      </div>
      <div className="text-center">
        <h2 className="text-2xl font-bold text-text-main mb-2">{title}</h2>
        <p className="text-text-muted max-w-md">
          {description || "This page is being ported from the Next.js dashboard. It will be available in a future update."}
        </p>
      </div>
    </div>
  );
}
