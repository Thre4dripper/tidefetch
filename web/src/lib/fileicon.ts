// Maps a download to a file-type icon chip (icon + tint) for task rows.
import {
  Disc3,
  FileArchive,
  FileAudio,
  FileCode2,
  FileImage,
  FileText,
  FileVideo,
  File,
  Magnet,
  Package
} from '@lucide/svelte';
import type { Component } from 'svelte';

export interface FileKind {
  icon: Component<{ size?: number | string; strokeWidth?: number | string }>;
  tint: string;
  label: string;
}

const kinds: { exts: string[]; kind: FileKind }[] = [
  { exts: ['iso', 'img', 'dmg', 'vhd', 'qcow2'], kind: { icon: Disc3, tint: 'cyan', label: 'Disk image' } },
  { exts: ['zip', 'tar', 'gz', 'tgz', 'bz2', 'xz', 'zst', '7z', 'rar'], kind: { icon: FileArchive, tint: 'cyan', label: 'Archive' } },
  { exts: ['mp4', 'mkv', 'webm', 'avi', 'mov', 'ts', 'm4v'], kind: { icon: FileVideo, tint: 'violet', label: 'Video' } },
  { exts: ['mp3', 'flac', 'ogg', 'wav', 'm4a', 'opus'], kind: { icon: FileAudio, tint: 'violet', label: 'Audio' } },
  { exts: ['jpg', 'jpeg', 'png', 'gif', 'webp', 'avif', 'svg', 'raw'], kind: { icon: FileImage, tint: 'amber', label: 'Image' } },
  { exts: ['pdf', 'epub', 'txt', 'md', 'doc', 'docx'], kind: { icon: FileText, tint: 'amber', label: 'Document' } },
  { exts: ['deb', 'rpm', 'apk', 'pkg', 'msi', 'exe', 'appimage', 'flatpak'], kind: { icon: Package, tint: 'lime', label: 'Package' } },
  { exts: ['js', 'ts', 'go', 'py', 'rs', 'c', 'cpp', 'java', 'json', 'yaml', 'yml'], kind: { icon: FileCode2, tint: 'lime', label: 'Source' } }
];

const byExt = new Map<string, FileKind>();
for (const group of kinds) {
  for (const ext of group.exts) byExt.set(ext, group.kind);
}

export function fileKind(name: string, torrent: boolean): FileKind {
  if (torrent || name.startsWith('magnet:')) {
    return { icon: Magnet, tint: 'lime', label: 'BitTorrent' };
  }
  const dot = name.lastIndexOf('.');
  const ext = dot >= 0 ? name.slice(dot + 1).toLowerCase() : '';
  return byExt.get(ext) ?? { icon: File, tint: 'dim', label: 'File' };
}
