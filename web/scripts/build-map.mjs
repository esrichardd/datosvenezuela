// Converts the simplified, projected admin1 boundaries (GeoJSON written by
// mapshaper) into the compact map data the site uses: one SVG path per
// state, plus a point to place its label. See src/data/README.md.
//
// Usage: node scripts/build-map.mjs <input.geojson> <output.json>
import { readFileSync, writeFileSync } from "node:fs";
import process from "node:process";

const [input, output] = process.argv.slice(2);
if (!input || !output) {
  process.stderr.write("Usage: node scripts/build-map.mjs <input> <output>\n");
  process.exit(1);
}

const WIDTH = 1000; // viewBox width; the height keeps the proportions
const MIN_ISLAND = 0.002; // drop polygons smaller than this share of the largest

const geojson = JSON.parse(readFileSync(input, "utf8"));
const polygonsOf = (geometry) =>
  geometry.type === "Polygon" ? [geometry.coordinates] : geometry.coordinates;

let [minX, minY, maxX, maxY] = [Infinity, Infinity, -Infinity, -Infinity];
for (const feature of geojson.features) {
  for (const polygon of polygonsOf(feature.geometry)) {
    for (const [x, y] of polygon.flat()) {
      minX = Math.min(minX, x);
      maxX = Math.max(maxX, x);
      minY = Math.min(minY, y);
      maxY = Math.max(maxY, y);
    }
  }
}

const scale = WIDTH / (maxX - minX);
const height = Math.round((maxY - minY) * scale);
const round = (n) => Math.round(n * 10) / 10;
// SVG's y axis points down, so the map is flipped vertically.
const project = ([x, y]) => [
  round((x - minX) * scale),
  round((maxY - y) * scale),
];

const area = (ring) => {
  let sum = 0;
  for (let i = 0; i < ring.length - 1; i++) {
    sum += ring[i][0] * ring[i + 1][1] - ring[i + 1][0] * ring[i][1];
  }
  return sum / 2;
};

const centroid = (ring) => {
  const a = area(ring);
  let cx = 0;
  let cy = 0;
  for (let i = 0; i < ring.length - 1; i++) {
    const c = ring[i][0] * ring[i + 1][1] - ring[i + 1][0] * ring[i][1];
    cx += (ring[i][0] + ring[i + 1][0]) * c;
    cy += (ring[i][1] + ring[i + 1][1]) * c;
  }
  return [Math.round(cx / (6 * a)), Math.round(cy / (6 * a))];
};

const toPath = (ring) => {
  const points = [];
  for (const point of ring.slice(0, -1)) {
    const last = points.at(-1);
    if (!last || last[0] !== point[0] || last[1] !== point[1])
      points.push(point);
  }
  return points.length < 3
    ? ""
    : `M${points.map(([x, y]) => `${x} ${y}`).join("L")}Z`;
};

const states = geojson.features
  .map((feature) => {
    const { adm1_pcode: code, adm1_name: name } = feature.properties;
    const polygons = polygonsOf(feature.geometry).map((polygon) =>
      polygon.map((ring) => ring.map(project)),
    );
    const largest = polygons.reduce((a, b) =>
      Math.abs(area(b[0])) > Math.abs(area(a[0])) ? b : a,
    );
    const minArea = Math.max(4, Math.abs(area(largest[0])) * MIN_ISLAND);
    const d = polygons
      .filter((p) => p === largest || Math.abs(area(p[0])) >= minArea)
      .flatMap((polygon) => polygon.map(toPath))
      .join("");
    return { code, name, d, label: centroid(largest[0]) };
  })
  .sort((a, b) => a.code.localeCompare(b.code));

writeFileSync(
  output,
  JSON.stringify({ viewBox: [0, 0, WIDTH, height], states }) + "\n",
);
process.stdout.write(`Wrote ${states.length} states to ${output}\n`);
