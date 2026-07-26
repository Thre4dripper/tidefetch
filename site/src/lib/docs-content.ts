// The documentation shown on the site is the repository's docs/ folder,
// imported at build time so there is exactly one source of truth.
const sources = import.meta.glob('../../../docs/**/*.md', {
  query: '?raw',
  import: 'default',
  eager: true
}) as Record<string, string>;

export type DocPage = {
  slug: string;
  title: string;
  description: string;
  source: string;
};

export type DocSection = { label: string; pages: DocPage[] };

function sourceFor(file: string): string {
  const key = `../../../docs/${file}`;
  return sources[key] ?? `# Not found\n\nMissing documentation file: \`docs/${file}\`.`;
}

const page = (slug: string, title: string, description: string, file: string): DocPage => ({
  slug,
  title,
  description,
  source: sourceFor(file)
});

export const docSections: DocSection[] = [
  {
    label: 'Introduction',
    pages: [
      page('getting-started', 'Getting Started', 'What Tidefetch is and the fastest ways to run it.', 'index.md'),
      page('installation', 'Installation', 'Source builds, containers, release archives and package managers.', 'installation.md')
    ]
  },
  {
    label: 'Guides',
    pages: [
      page('configuration', 'Configuration', 'Commands, flags, files, environment variables and authentication.', 'configuration.md'),
      page('homelab', 'Homelab Operations', 'Storage, networking, backups, upgrades and NAS platforms.', 'homelab.md'),
      page('reverse-proxy', 'Reverse Proxy & TLS', 'Caddy, Nginx, Traefik, Kubernetes Ingress and Tailscale.', 'reverse-proxy.md'),
      page('troubleshooting', 'Troubleshooting', 'Diagnostics, common failures and safe recovery.', 'troubleshooting.md')
    ]
  },
  {
    label: 'Deployment',
    pages: [
      page('deployment/docker', 'Docker & Podman', 'The all-in-one container with volumes and secrets.', 'deployment/docker.md'),
      page('deployment/swarm', 'Docker Swarm', 'Single-replica stack with Docker secrets.', 'deployment/swarm.md'),
      page('deployment/kubernetes', 'Kubernetes & k3s', 'Manifests, PVCs, probes and Ingress.', 'deployment/kubernetes.md'),
      page('deployment/unraid', 'Unraid', 'Community Applications template and shares.', 'deployment/unraid.md')
    ]
  }
];

export const docPages: DocPage[] = docSections.flatMap((section) => section.pages);

export function findDoc(slug: string): DocPage | undefined {
  return docPages.find((p) => p.slug === slug);
}

export function adjacentDocs(slug: string): { prev?: DocPage; next?: DocPage } {
  const index = docPages.findIndex((p) => p.slug === slug);
  if (index === -1) return {};
  return { prev: docPages[index - 1], next: docPages[index + 1] };
}
