// Text, Byte, and BBCode Formatting Utilities

export function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function escapeHtml(str) {
  if (!str) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

export function formatTimeAgo(ts) {
  if (!ts) return '';
  const tsMs = ts > 1e11 ? ts : ts * 1000;
  const sec = Math.floor((Date.now() - tsMs) / 1000);
  if (sec < 60) return 'Just now';
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  return new Date(tsMs).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

export function formatNewsHTML(raw) {
  if (!raw) return '';

  let s = String(raw).trim();

  // 1. Unescape common HTML entities
  s = s.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"');

  // 2. Normalize Steam Clan Image tokens & fix double slashes
  s = s.replace(/\{STEAM_CLAN_IMAGE\}/g, 'https://clan.cloudflare.steamstatic.com/images/');
  s = s.replace(/\{STEAM_CLAN_LOC_IMAGE\}/g, 'https://clan.cloudflare.steamstatic.com/images/');
  s = s.replace(/images\/\/+/g, 'images/');

  // 3. YouTube Previews
  s = s.replace(/\[previewyoutube=([a-zA-Z0-9_-]+);?[^\]]*\]\[\/previewyoutube\]/gi, (_m, id) => {
    return `<div class="my-3 rounded-xl overflow-hidden border border-[#2b2b3c] bg-black/40"><a href="https://www.youtube.com/watch?v=${id}" target="_blank" rel="noopener noreferrer" class="block relative group"><img src="https://img.youtube.com/vi/${id}/hqdefault.jpg" alt="YouTube Preview" class="w-full max-h-56 object-cover" /><div class="absolute inset-0 flex items-center justify-center bg-black/30 group-hover:bg-black/10 transition-colors"><span class="bg-red-600 text-white text-xs font-bold px-3 py-1.5 rounded-lg shadow-lg">▶ Watch on YouTube</span></div></a></div>`;
  });

  // 4. Dynamic Links
  s = s.replace(/\[dynamiclink\s+href=["']?([^"']+)["']?[^\]]*\](.*?)\[\/dynamiclink\]/gis, (_m, href, inner) => {
    const label = inner.trim() || href;
    return `<a href="${href}" target="_blank" rel="noopener noreferrer" class="inline-flex items-center space-x-1 text-sky-400 hover:text-sky-300 underline font-medium my-1"><span>${label}</span><span>↗</span></a>`;
  });

  // 5. Images (both [img src="..."][/img] and [img]...[/img])
  s = s.replace(/\[img\s+src=["']?([^"'\s\]]+)["']?[^\]]*\](?:\[\/img\])?/gis, '<img src="$1" alt="" class="my-3 rounded-xl max-h-72 w-full object-cover border border-[#2b2b3c] bg-black/30" loading="lazy" />');
  s = s.replace(/\[img[^\]]*\](.*?)\[\/img\]/gis, '<img src="$1" alt="" class="my-3 rounded-xl max-h-72 w-full object-cover border border-[#2b2b3c] bg-black/30" loading="lazy" />');

  // 6. Headers (BBCode & HTML)
  s = s.replace(/\[h1\](.*?)\[\/h1\]/gis, '<h3 class="text-sm font-black text-white mt-4 mb-2 pb-1 border-b border-[#252535]">$1</h3>');
  s = s.replace(/\[h2\](.*?)\[\/h2\]/gis, '<h4 class="text-xs font-bold text-slate-100 mt-3 mb-1.5">$1</h4>');
  s = s.replace(/\[h3\](.*?)\[\/h3\]/gis, '<h5 class="text-xs font-bold text-slate-200 mt-2.5 mb-1">$1</h5>');
  s = s.replace(/<h[1-6][^>]*>(.*?)<\/h[1-6]>/gis, '<h4 class="text-xs font-bold text-white mt-3 mb-1">$1</h4>');

  // 7. Paragraphs & Line Breaks
  s = s.replace(/\[p[^\]]*\]/gi, '<p class="my-2 leading-relaxed">').replace(/\[\/p\]/gi, '</p>');
  s = s.replace(/<p[^>]*>/gi, '<p class="my-2 leading-relaxed">').replace(/<\/p>/gi, '</p>');
  s = s.replace(/\[br\]/gi, '<br/>').replace(/<br\s*\/?>/gi, '<br/>');
  s = s.replace(/\[hr\]/gi, '<hr class="border-[#262638] my-3" />').replace(/<hr\s*\/?>/gi, '<hr class="border-[#262638] my-3" />');

  // 8. Formatting tags
  s = s.replace(/\[b\](.*?)\[\/b\]/gis, '<strong class="font-bold text-slate-100">$1</strong>');
  s = s.replace(/<strong[^>]*>(.*?)<\/strong>/gis, '<strong class="font-bold text-slate-100">$1</strong>');
  s = s.replace(/<b[^>]*>(.*?)<\/b>/gis, '<strong class="font-bold text-slate-100">$1</strong>');

  s = s.replace(/\[i\](.*?)\[\/i\]/gis, '<em class="italic text-slate-300">$1</em>');
  s = s.replace(/<em[^>]*>(.*?)<\/em>/gis, '<em class="italic text-slate-300">$1</em>');
  s = s.replace(/<i[^>]*>(.*?)<\/i>/gis, '<em class="italic text-slate-300">$1</em>');

  s = s.replace(/\[u\](.*?)\[\/u\]/gis, '<u class="underline decoration-slate-500">$1</u>');
  s = s.replace(/<u[^>]*>(.*?)<\/u>/gis, '<u class="underline decoration-slate-500">$1</u>');

  s = s.replace(/\[strike\](.*?)\[\/strike\]/gis, '<del class="line-through text-slate-500">$1</del>');
  s = s.replace(/<del[^>]*>(.*?)<\/del>/gis, '<del class="line-through text-slate-500">$1</del>');

  s = s.replace(/\[code\](.*?)\[\/code\]/gis, '<pre class="bg-[#0b0b10] border border-[#242434] text-slate-300 p-2.5 rounded-xl font-mono text-[11px] my-2 overflow-x-auto"><code>$1</code></pre>');
  s = s.replace(/\[quote\](.*?)\[\/quote\]/gis, '<blockquote class="border-l-2 border-slate-600 pl-3 my-2 text-slate-400 italic">$1</blockquote>');

  // 9. Lists & List item closers ([/*] cleanup)
  s = s.replace(/\[list\](.*?)\[\/list\]/gis, (_m, listBody) => {
    const cleanBody = listBody.replace(/\[\/\*\]/gi, '');
    const items = cleanBody.split(/\[\*\]/).filter(it => it.trim().length > 0);
    if (items.length > 0) {
      const lis = items.map(it => `<li class="ml-4 pl-1 text-slate-300 leading-relaxed">${it.trim()}</li>`).join('');
      return `<ul class="list-disc space-y-1 my-2 text-xs">${lis}</ul>`;
    }
    return listBody;
  });
  s = s.replace(/\[olist\](.*?)\[\/olist\]/gis, (_m, listBody) => {
    const cleanBody = listBody.replace(/\[\/\*\]/gi, '');
    const items = cleanBody.split(/\[\*\]/).filter(it => it.trim().length > 0);
    if (items.length > 0) {
      const lis = items.map(it => `<li class="ml-4 pl-1 text-slate-300 leading-relaxed">${it.trim()}</li>`).join('');
      return `<ol class="list-decimal space-y-1 my-2 text-xs">${lis}</ol>`;
    }
    return listBody;
  });
  s = s.replace(/<ul[^>]*>/gi, '<ul class="list-disc space-y-1 my-2 text-xs">').replace(/<\/ul>/gi, '</ul>');
  s = s.replace(/<ol[^>]*>/gi, '<ol class="list-decimal space-y-1 my-2 text-xs">').replace(/<\/ol>/gi, '</ol>');
  s = s.replace(/<li[^>]*>/gi, '<li class="ml-4 pl-1 text-slate-300 leading-relaxed">').replace(/<\/li>/gi, '</li>');
  s = s.replace(/\[\/\*\]/gi, ''); // Stray list closers

  // 10. Links (standard [url] and [url=...])
  s = s.replace(/\[url=([^\]]+)\](.*?)\[\/url\]/gis, '<a href="$1" target="_blank" rel="noopener noreferrer" class="text-sky-400 hover:text-sky-300 underline font-medium">$2 ↗</a>');
  s = s.replace(/\[url\](.*?)\[\/url\]/gis, '<a href="$1" target="_blank" rel="noopener noreferrer" class="text-sky-400 hover:text-sky-300 underline font-medium">$1 ↗</a>');

  // 11. Catch-all: Strip any remaining unhandled BBCode or HTML containers safely
  s = s.replace(/\[\/?(?:table|tr|th|td|spoiler|expand|section|docref|dynamiclink|strike|noparse|align)[^\]]*\]/gi, '');
  s = s.replace(/\[\/?[a-zA-Z0-9_-]+(?:=[^\]]+|\s+[^\]]+)?\]/gi, '');
  s = s.replace(/<\/?(?:div|span|table|tr|th|td|section|article)[^>]*>/gi, '');

  // 12. Clean up newlines and empty elements
  s = s.replace(/\n\s*\n/g, '<br/><br/>');
  s = s.replace(/\n/g, '<br/>');
  s = s.replace(/<p class="[^"]*"><\/p>/gi, '');
  s = s.replace(/(?:<br\s*\/?>\s*){3,}/gi, '<br/><br/>');

  return s;
}

export function cleanBBCode(bbcode) {
  return formatNewsHTML(bbcode);
}
