(() => {
  const fixtureInfo = {
    valid: true,
    grid_size: [57584.11328125, 64194.1953125],
    grid_steps: [6400, 6400],
    grid_zero: [-24816.11328125, 31426.1953125],
    map_min: [-32768, -32768],
    map_max: [32768, 32768],
    map_generation: 1
  };

  const fixtureObjects = [
    { type: "airfield", color: "#174DFF", sx: 0.359126, sy: 0.560636, ex: 0.359155, ey: 0.511808 },
    { type: "aircraft", color: "#faC81E", icon: "Player", x: 0.360958, y: 0.353662, dx: -0.002011, dy: -0.999998 },
    { type: "aircraft", color: "#f00C00", icon: "Fighter", x: 0.314544, y: 0.522145, dx: -0.911217, dy: -0.411928 },
    { type: "aircraft", color: "#f00C00", icon: "Fighter", x: 0.435529, y: 0.498517, dx: 0.015462, dy: -0.99988 },
    { type: "aircraft", color: "#f00C00", icon: "Fighter", x: 0.314258, y: 0.130046, dx: 0.19342, dy: 0.981116 },
    { type: "aircraft", color: "#f00C00", icon: "Fighter", x: 0.256025, y: 0.170275, dx: -0.045846, dy: 0.998949 },
    { type: "ground_model", color: "#fa0C00", icon: "LightTank", x: 0.354096, y: 0.488192 },
    { type: "ground_model", color: "#fa0C00", icon: "LightTank", x: 0.360949, y: 0.488621 },
    { type: "ground_model", color: "#fa0C00", icon: "MediumTank", x: 0.354317, y: 0.48466 },
    { type: "ground_model", color: "#fa0C00", icon: "MediumTank", x: 0.358124, y: 0.484899 },
    { type: "ground_model", color: "#fa0C00", icon: "SPAA", x: 0.327647, y: 0.445153 },
    { type: "ground_model", color: "#fa0C00", icon: "SAM", x: 0.290852, y: 0.443951 },
    { type: "ground_model", color: "#fa0C00", icon: "SAM", x: 0.756714, y: 0.582993 },
    { type: "bombing_point", color: "#fa0C00", icon: "bombing_point", x: 0.427345, y: 0.50683 },
    { type: "bombing_point", color: "#fa0C00", icon: "bombing_point", x: 0.324332, y: 0.524698 },
    { type: "bombing_point", color: "#fa0C00", icon: "bombing_point", x: 0.387898, y: 0.674336 },
    { type: "bombing_point", color: "#fa0C00", icon: "bombing_point", x: 0.298854, y: 0.585138 }
  ];

  const connection = document.querySelector("[data-connection]");
  const connectionDetail = connection?.querySelector("span");

  function setConnection(state, label, detail) {
    if (!connection) return;
    connection.dataset.state = state;
    connection.firstChild.textContent = `${label} `;
    if (connectionDetail) connectionDetail.textContent = detail;
  }

  class TacticalMap {
    constructor(canvas) {
      this.canvas = canvas;
      this.context = canvas.getContext("2d");
      this.mode = canvas.dataset.mapMode || "full";
      this.mapInfo = fixtureInfo;
      this.objects = fixtureObjects;
      this.image = null;
      this.resizeObserver = new ResizeObserver(() => this.resize());
      this.resizeObserver.observe(canvas);
      this.resize();
      this.loadFixture();
    }

    async loadFixture() {
      try {
        const response = await fetch("/docs/fixtures/air-test-flight-jh-7/map-objects.json", { cache: "no-store" });
        if (!response.ok) throw new Error(`Fixture HTTP ${response.status}`);
        this.objects = await response.json();
      } catch {
        setConnection("stale", "Partial fixture", "inline 17-object fallback");
      }
      this.updateCounts();
      this.draw();
    }

    updateCounts(viewport = { x: 0, y: 0, size: 1 }) {
      const isVisible = (item) => {
        const x = item.type === "airfield" ? (item.sx + item.ex) / 2 : item.x;
        const y = item.type === "airfield" ? (item.sy + item.ey) / 2 : item.y;
        return Number.isFinite(x) && Number.isFinite(y) &&
          x >= viewport.x && y >= viewport.y &&
          x <= viewport.x + viewport.size && y <= viewport.y + viewport.size;
      };
      const format = (items) => {
        const visible = items.filter(isVisible).length;
        return viewport.size < 0.999 ? `${visible} / ${items.length}` : String(items.length);
      };
      const hostileAir = this.objects.filter((item) => item.type === "aircraft" && item.icon !== "Player");
      const ground = this.objects.filter((item) => item.type === "ground_model");
      const airfields = this.objects.filter((item) => item.type === "airfield");
      const airDefense = this.objects.filter((item) => item.icon === "SAM" || item.icon === "SPAA");
      const targets = this.objects.filter((item) => item.icon === "bombing_point");
      document.querySelectorAll("[data-count-air]").forEach((node) => { node.textContent = format(hostileAir); });
      document.querySelectorAll("[data-count-ground]").forEach((node) => { node.textContent = format(ground); });
      document.querySelectorAll("[data-count-airfield]").forEach((node) => { node.textContent = format(airfields); });
      document.querySelectorAll("[data-count-airdef]").forEach((node) => { node.textContent = format(airDefense); });
      document.querySelectorAll("[data-count-target]").forEach((node) => { node.textContent = format(targets); });
      const visibleTotal = this.objects.filter(isVisible).length;
      document.querySelectorAll("[data-count-total]").forEach((node) => {
        node.textContent = viewport.size < 0.999 ? `${visibleTotal} / ${this.objects.length} visible` : String(this.objects.length);
      });
    }

    resize() {
      const rect = this.canvas.getBoundingClientRect();
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      const width = Math.max(1, Math.round(rect.width * ratio));
      const height = Math.max(1, Math.round(rect.height * ratio));
      if (this.canvas.width !== width || this.canvas.height !== height) {
        this.canvas.width = width;
        this.canvas.height = height;
      }
      this.draw();
    }

    getMapRect() {
      const size = Math.min(this.canvas.width, this.canvas.height);
      return {
        x: Math.round((this.canvas.width - size) / 2),
        y: Math.round((this.canvas.height - size) / 2),
        size
      };
    }

    getViewport() {
      if (this.mode === "full") return { x: 0, y: 0, size: 1 };
      const player = this.objects.find((item) => item.icon === "Player");
      if (!player) return { x: 0, y: 0, size: 1 };

      const candidates = this.objects
        .filter((item) => Number.isFinite(item.x) && Number.isFinite(item.y) && item.icon !== "Player")
        .map((item) => ({ ...item, distance: Math.hypot(item.x - player.x, item.y - player.y) }))
        .sort((a, b) => a.distance - b.distance);

      const points = [player];
      const destination = candidates.find((item) => item.type === "bombing_point") || candidates[0];
      if (this.mode === "threat") {
        points.push(...candidates.filter((item) => item.type === "aircraft").slice(0, 3));
        if (destination) points.push(destination);
      } else {
        if (destination) points.push(destination);
        points.push(...candidates.filter((item) => item.distance < 0.16).slice(0, 5));
      }

      const minX = Math.min(...points.map((point) => point.x));
      const maxX = Math.max(...points.map((point) => point.x));
      const minY = Math.min(...points.map((point) => point.y));
      const maxY = Math.max(...points.map((point) => point.y));
      const size = Math.min(1, Math.max(0.28, Math.max(maxX - minX, maxY - minY) * 1.45));
      const centerX = (minX + maxX) / 2;
      const centerY = (minY + maxY) / 2;
      return {
        x: Math.max(0, Math.min(1 - size, centerX - size / 2)),
        y: Math.max(0, Math.min(1 - size, centerY - size / 2)),
        size
      };
    }

    toScreen(valueX, valueY, rect, viewport) {
      return {
        x: rect.x + ((valueX - viewport.x) / viewport.size) * rect.size,
        y: rect.y + ((valueY - viewport.y) / viewport.size) * rect.size
      };
    }

    draw() {
      const ctx = this.context;
      const width = this.canvas.width;
      const height = this.canvas.height;
      if (!width || !height) return;
      const rect = this.getMapRect();
      const viewport = this.getViewport();
      this.updateCounts(viewport);

      ctx.clearRect(0, 0, width, height);
      ctx.fillStyle = "#121517";
      ctx.fillRect(0, 0, width, height);
      ctx.save();
      ctx.beginPath();
      ctx.rect(rect.x, rect.y, rect.size, rect.size);
      ctx.clip();
      if (this.image?.complete && this.image.naturalWidth) {
        const sourceSize = this.image.naturalWidth * viewport.size;
        ctx.drawImage(
          this.image,
          this.image.naturalWidth * viewport.x,
          this.image.naturalHeight * viewport.y,
          sourceSize,
          sourceSize,
          rect.x,
          rect.y,
          rect.size,
          rect.size
        );
      } else {
        this.drawPlaceholder(ctx, rect);
      }
      this.drawScanTexture(ctx, rect);
      this.drawGrid(ctx, rect, viewport);
      this.objects
        .filter((object) => object.type !== "ground_model" && object.type !== "aircraft")
        .forEach((object) => this.drawObject(ctx, rect, viewport, object));
      this.drawGroundClusters(ctx, rect, viewport);
      this.drawNavigation(ctx, rect, viewport);
      this.objects
        .filter((object) => object.type === "aircraft")
        .forEach((object) => this.drawObject(ctx, rect, viewport, object));
      ctx.restore();
    }

    drawPlaceholder(ctx, rect) {
      ctx.fillStyle = "#404040";
      ctx.fillRect(rect.x, rect.y, rect.size, rect.size);
      ctx.save();
      ctx.globalAlpha = 0.42;
      ctx.fillStyle = "#808080";
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.font = `900 ${rect.size * 0.55}px Arial`;
      ctx.fillText("?", rect.x + rect.size * 0.5, rect.y + rect.size * 0.51);
      ctx.restore();
    }

    drawGrid(ctx, rect, viewport) {
      const { map_min: min, map_max: max, grid_steps: steps } = this.mapInfo;
      if (!min || !max || !steps) return;
      const worldWidth = max[0] - min[0];
      const worldHeight = max[1] - min[1];
      const scaleX = rect.size / viewport.size / worldWidth;
      const scaleY = rect.size / viewport.size / worldHeight;
      const firstColumn = Math.floor(viewport.x * worldWidth / steps[0]);
      const firstRow = Math.floor(viewport.y * worldHeight / steps[1]);
      const lastColumn = Math.ceil((viewport.x + viewport.size) * worldWidth / steps[0]);
      const lastRow = Math.ceil((viewport.y + viewport.size) * worldHeight / steps[1]);

      ctx.save();
      ctx.strokeStyle = "rgba(255, 255, 255, 0.16)";
      ctx.fillStyle = "rgba(235, 238, 236, 0.9)";
      ctx.shadowColor = "rgba(0, 0, 0, 0.95)";
      ctx.shadowBlur = 3;
      ctx.lineWidth = Math.max(1, window.devicePixelRatio || 1);
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      ctx.font = `700 ${Math.max(11 * ratio, Math.round(rect.size * 0.018))}px Arial`;

      for (let row = firstRow; row <= lastRow; row += 1) {
        const normalizedY = (row * steps[1]) / worldHeight;
        const screen = this.toScreen(0, normalizedY, rect, viewport);
        ctx.beginPath();
        ctx.moveTo(rect.x, screen.y);
        ctx.lineTo(rect.x + rect.size, screen.y);
        ctx.stroke();
        if (row >= 0) {
          ctx.textAlign = "left";
          ctx.textBaseline = "middle";
          ctx.fillText(this.rowLabel(row), rect.x + 5, screen.y + steps[1] * scaleY * 0.5);
        }
      }

      for (let column = firstColumn; column <= lastColumn; column += 1) {
        const normalizedX = (column * steps[0]) / worldWidth;
        const screen = this.toScreen(normalizedX, 0, rect, viewport);
        ctx.beginPath();
        ctx.moveTo(screen.x, rect.y);
        ctx.lineTo(screen.x, rect.y + rect.size);
        ctx.stroke();
        if (column >= 0) {
          ctx.textAlign = "center";
          ctx.textBaseline = "top";
          ctx.fillText(String(column + 1), screen.x + steps[0] * scaleX * 0.5, rect.y + 5);
        }
      }
      ctx.restore();
    }

    rowLabel(index) {
      let label = "";
      let value = index + 1;
      while (value > 0) {
        value -= 1;
        label = String.fromCharCode(65 + (value % 26)) + label;
        value = Math.floor(value / 26);
      }
      return label;
    }

    drawObject(ctx, rect, viewport, object) {
      if (object.type === "airfield") {
        if (![object.sx, object.sy, object.ex, object.ey].every(Number.isFinite)) return;
        const start = this.toScreen(object.sx, object.sy, rect, viewport);
        const end = this.toScreen(object.ex, object.ey, rect, viewport);
        ctx.save();
        ctx.strokeStyle = object.color || "#174DFF";
        ctx.lineWidth = Math.max(3, Math.sqrt(rect.size / 640) * 3);
        ctx.beginPath();
        ctx.moveTo(start.x, start.y);
        ctx.lineTo(end.x, end.y);
        ctx.stroke();
        ctx.restore();
        return;
      }

      if (!Number.isFinite(object.x) || !Number.isFinite(object.y)) return;
      if (
        object.x < viewport.x ||
        object.y < viewport.y ||
        object.x > viewport.x + viewport.size ||
        object.y > viewport.y + viewport.size
      ) return;

      const point = this.toScreen(object.x, object.y, rect, viewport);
      if (object.icon === "Player") {
        this.drawPlayer(ctx, point.x, point.y, object, rect.size);
        return;
      }

      ctx.save();
      ctx.translate(point.x, point.y);
      ctx.fillStyle = object.color || "#f00C00";
      ctx.strokeStyle = "#080808";
      ctx.lineWidth = Math.max(1, window.devicePixelRatio || 1);
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      const size = Math.max(8 * ratio, rect.size * 0.014);

      if (object.type === "aircraft") {
        ctx.rotate(Math.atan2(object.dx || 0, -(object.dy ?? -1)));
        ctx.beginPath();
        ctx.moveTo(0, -size);
        ctx.lineTo(size * 0.35, size * 0.3);
        ctx.lineTo(size, size * 0.65);
        ctx.lineTo(size * 0.2, size * 0.5);
        ctx.lineTo(0, size);
        ctx.lineTo(-size * 0.2, size * 0.5);
        ctx.lineTo(-size, size * 0.65);
        ctx.lineTo(-size * 0.35, size * 0.3);
        ctx.closePath();
      } else if (object.icon === "bombing_point") {
        ctx.beginPath();
        ctx.arc(0, 0, size * 0.8, 0, Math.PI * 2);
        ctx.moveTo(-size, 0);
        ctx.lineTo(size, 0);
        ctx.moveTo(0, -size);
        ctx.lineTo(0, size);
      } else if (object.icon === "SAM" || object.icon === "SPAA") {
        ctx.beginPath();
        ctx.moveTo(0, -size);
        ctx.lineTo(size, size);
        ctx.lineTo(-size, size);
        ctx.closePath();
      } else if (object.type === "ground_model") {
        ctx.beginPath();
        ctx.rect(-size * 0.65, -size * 0.65, size * 1.3, size * 1.3);
      } else {
        ctx.beginPath();
        ctx.moveTo(0, -size);
        ctx.lineTo(size, 0);
        ctx.lineTo(0, size);
        ctx.lineTo(-size, 0);
        ctx.closePath();
      }
      ctx.fill();
      ctx.stroke();
      ctx.restore();
    }

    drawGroundClusters(ctx, rect, viewport) {
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      const mergeDistance = 14 * ratio;
      const clusters = [];

      this.objects
        .filter((object) => object.type === "ground_model" && Number.isFinite(object.x) && Number.isFinite(object.y))
        .forEach((object) => {
          if (
            object.x < viewport.x ||
            object.y < viewport.y ||
            object.x > viewport.x + viewport.size ||
            object.y > viewport.y + viewport.size
          ) return;
          const point = this.toScreen(object.x, object.y, rect, viewport);
          const existing = clusters.find((cluster) => Math.hypot(cluster.x - point.x, cluster.y - point.y) < mergeDistance);
          if (existing) {
            existing.items.push(object);
            existing.x = (existing.x * (existing.items.length - 1) + point.x) / existing.items.length;
            existing.y = (existing.y * (existing.items.length - 1) + point.y) / existing.items.length;
          } else {
            clusters.push({ x: point.x, y: point.y, items: [object] });
          }
        });

      clusters.forEach((cluster) => {
        if (cluster.items.length === 1) {
          this.drawObject(ctx, rect, viewport, cluster.items[0]);
          return;
        }
        const radius = Math.max(10 * ratio, rect.size * 0.018);
        ctx.save();
        ctx.fillStyle = cluster.items[0].color || "#fa0C00";
        ctx.strokeStyle = "#080808";
        ctx.lineWidth = 2 * ratio;
        ctx.beginPath();
        ctx.arc(cluster.x, cluster.y, radius, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        ctx.fillStyle = "#fff";
        ctx.font = `800 ${10 * ratio}px Arial`;
        ctx.textAlign = "center";
        ctx.textBaseline = "middle";
        ctx.fillText(String(cluster.items.length), cluster.x, cluster.y);
        ctx.restore();
      });
    }

    drawNavigation(ctx, rect, viewport) {
      const player = this.objects.find((item) => item.icon === "Player");
      if (!player) return;
      const destination = this.objects
        .filter((item) => item.type === "bombing_point")
        .sort((a, b) =>
          Math.hypot(a.x - player.x, a.y - player.y) -
          Math.hypot(b.x - player.x, b.y - player.y)
        )[0];
      if (!destination) return;
      if (
        destination.x < viewport.x ||
        destination.y < viewport.y ||
        destination.x > viewport.x + viewport.size ||
        destination.y > viewport.y + viewport.size
      ) return;

      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      const start = this.toScreen(player.x, player.y, rect, viewport);
      const end = this.toScreen(destination.x, destination.y, rect, viewport);
      const radius = 11 * ratio;
      ctx.save();
      ctx.strokeStyle = "#f4b13d";
      ctx.lineWidth = 2 * ratio;
      ctx.setLineDash([6 * ratio, 6 * ratio]);
      ctx.beginPath();
      ctx.moveTo(start.x, start.y);
      ctx.lineTo(end.x, end.y);
      ctx.stroke();
      ctx.setLineDash([]);
      ctx.beginPath();
      ctx.arc(end.x, end.y, radius, 0, Math.PI * 2);
      ctx.stroke();
      ctx.beginPath();
      ctx.moveTo(end.x - radius * 1.4, end.y);
      ctx.lineTo(end.x + radius * 1.4, end.y);
      ctx.moveTo(end.x, end.y - radius * 1.4);
      ctx.lineTo(end.x, end.y + radius * 1.4);
      ctx.stroke();
      ctx.restore();
    }

    drawPlayer(ctx, x, y, object, mapSize) {
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      const length = Math.max(22 * ratio, mapSize * 0.04);
      const width = length * 0.28;
      ctx.save();
      ctx.translate(x, y);
      ctx.rotate(Math.atan2(object.dx || 0, -(object.dy ?? -1)));
      ctx.fillStyle = "#ffffff";
      ctx.strokeStyle = "#262626";
      ctx.lineWidth = Math.max(2, window.devicePixelRatio || 1);
      ctx.beginPath();
      ctx.moveTo(0, -length * 0.6);
      ctx.lineTo(width, length * 0.42);
      ctx.lineTo(0, length * 0.2);
      ctx.lineTo(-width, length * 0.42);
      ctx.closePath();
      ctx.fill();
      ctx.stroke();
      ctx.restore();
    }

    drawScanTexture(ctx, rect) {
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      ctx.save();
      ctx.globalAlpha = 0.06;
      ctx.fillStyle = "#000";
      for (let y = rect.y; y < rect.y + rect.size; y += 4 * ratio) {
        ctx.fillRect(rect.x, y, rect.size, ratio);
      }
      ctx.restore();
    }
  }

  document.querySelectorAll("[data-tactical-map]").forEach((canvas) => new TacticalMap(canvas));
  setConnection("fixture", "Prototype", "captured map/telemetry · illustrative mission");

  document.querySelectorAll(".feed-panel").forEach((panel) => {
    const body = panel.querySelector(".feed-body");
    const rows = [...panel.querySelectorAll(".feed-row")];
    const status = panel.querySelector("[data-feed-status]");
    const updateCapacity = () => {
      rows.forEach((row) => { row.hidden = false; });
      let used = 0;
      let shown = 0;
      for (const row of rows) {
        const height = row.getBoundingClientRect().height;
        if (used + height > body.clientHeight && shown > 0) {
          row.hidden = true;
        } else {
          used += height;
          shown += 1;
        }
      }
      if (status) status.textContent = `${shown} shown · sample`;
    };
    new ResizeObserver(updateCapacity).observe(body);
    updateCapacity();
  });
})();
