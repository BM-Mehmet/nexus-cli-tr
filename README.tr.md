# Nexus-CLI (Go için DeepSeek Web Köprüsü) ⚡

[English](README.md) | [Türkçe](README.tr.md) | [Русский](README.ru.md) | [O'zbekcha](README.uz.md)

---

Ücretsiz web hesabı (`chat.deepseek.com`) üzerinden herhangi bir uygulamanın veya kodlama aracısının DeepSeek **V3** (`deepseek-chat`) ve **R1** (`deepseek-reasoner` / Thinking) modellerine bağlanmasını sağlayan, yüksek performanslı bir Go CLI aracı ve yerel API proxy sunucusudur. Resmî, ücretli DeepSeek API anahtarı gerekmez.

Node.js (`FreeDeepseekAPI-EN`) ve Python (`deepseek_bridge`) projelerinin ayrıntılı analizine dayanarak saf Go (Golang) ile baştan yazılmıştır; **çalışma zamanı bağımlılığı yoktur**.

---

## 🚀 Temel Özellikler

1. **Saf Go ile paralel PoW (Proof-of-Work) çözücüsü:**
   - CPU çekirdeklerine paralel çalışan `DeepSeekHashV1` algoritmasını (Keccak-f[1600], 1..23 tur SHA3-256) kullanarak DeepSeek sunucu sınamalarını çözer.
   - 144.000 zorluk düzeyindeki sınamaları yalnızca **~40–50 milisaniyede** çözer (CGO, WASM veya Python bağımlılığı gerekmez).
2. **Etkileşimli terminal sohbeti (REPL):**
   - R1 modelinin akıl yürütme sürecini (`reasoning_content`) özel biçimlendirmeyle (italik/camgöbeği) gerçek zamanlı gösterir.
   - Renkli istemlere ve animasyonlu yükleme göstergelerine sahip, özenle hazırlanmış ANSI arayüzü.
   - Yerleşik slash komutları: `/help`, `/new`, `/model`, `/thinking`, `/search`, `/whoami`, `/clear`, `/exit`.
3. **Tek seferlik istek ve Unix pipe desteği:**
   - Doğrudan CLI sorguları: `nexus "Kuantum hesaplama nedir?"`
   - Linux/macOS pipe iş akışları: `cat main.go | nexus "Bu kodu güvenlik ve performans açısından incele"`
4. **OpenAI, Anthropic ve Ollama uyumlu proxy sunucusu (`nexus serve`):**
   - `POST /v1/chat/completions` (OpenAI biçimi, SSE akışı ve JSON)
   - `POST /v1/messages` (Anthropic Messages API — **Claude Code** ve Anthropic SDK ile %100 uyumlu!)
   - `POST /v1/responses` (OpenAI Responses API)
   - `GET /v1/models` ve `GET /v1/model-capabilities`
   - `GET /v1/sessions` ve `POST /v1/sessions/reset`
   - Open WebUI, Continue.dev, Cursor, Cline ve diğer tüm OpenAI/Anthropic istemcileriyle sorunsuz çalışır.
5. **Yerel Ollama API benzetimi 🦙:**
   - `GET /api/tags`, `POST /api/chat`, `POST /api/generate`, `GET /api/version` uç noktalarını taklit eder.
   - Open WebUI, Dify, AnythingLLM, Cursor, LlamaIndex ve resmî Ollama SDK’larıyla doğrudan kullanıma hazır uyumluluk sağlar.
6. **Yapılandırılmış çıktılar ve JSON Schema 🎯:**
   - CLI, Ollama ve OpenAI uç noktalarında katı JSON çıktısı zorunlu kılar veya çıktıyı JSON Schema’ya göre doğrular.
7. **Evrensel tümleştirmeler (Launcher) 🚀:**
   - Claude Code, DeepSeek Harness, Codex, OpenCode, Goose ve Aider için tek komutla başlatıcı (Nexus proxy’yi arka planda otomatik başlatır).
8. **Çoklu hesap havuzu 👥:**
   - İstek sınırı engellerini (HTTP 429) ortadan kaldırmak için birden fazla DeepSeek hesabını Round-Robin zamanlamasıyla sırayla kullanır.
9. **Kimlik doğrulama yönetimi (`nexus auth`):**
   - Elle giriş: `nexus auth login` (tarayıcı LocalStorage alanından `userToken` değerini alır)
   - Chrome ile otomatik: `nexus auth chrome` (CDP aracılığıyla tarayıcı belleğinden oturum belirtecini okur)
   - Hesap bilgilerini görüntüleme: `nexus auth whoami`
10. **Sistem tanılaması (Doctor):**
    - `nexus doctor` — belirteç geçerliliğini, sunucu bağlantısını, PoW çözücü performansını ve oturum oluşturmayı anında denetler.

---

## 📦 Kurulum ve Derleme

Go 1.20 veya üzeri gerekir:

```bash
# Depoyu klonlayıp dizine geçin:
git clone https://github.com/xamrayev/nexus-cli.git
cd nexus-cli

# Çalıştırılabilir dosyayı derleyin:
go build -o bin/nexus .

# İsteğe bağlı olarak sistem PATH alanına kurun:
go install .
```

---

## 🔑 Kimlik Doğrulama (Hesap Bağlama)

DeepSeek hesabınızı bağlamanın 3 yolu vardır:

### 1. Etkileşimli CLI istemi:
```bash
./bin/nexus auth login
```
`userToken` değeriniz istenir:
1. Tarayıcınızda `https://chat.deepseek.com` adresini açın.
2. `F12` tuşuna (macOS’ta `Cmd+Option+I`) basın.
3. `Application` -> `Local Storage` -> `https://chat.deepseek.com` bölümüne gidin; `userToken` değerini kopyalayıp isteme yapıştırın.

### 2. Chrome ile otomatik (CDP):
```bash
./bin/nexus auth chrome
```
Bu komut otomatik olarak bir Chrome penceresi açar. Giriş yaptığınızda Nexus oturum belirtecini güvenli biçimde alıp kaydeder.

### 3. `.env` dosyasıyla:
Proje kök dizininde bir `.env` dosyası oluşturun:
```env
DEEPSEEK_TOKEN=your_deepseek_token_here
```

---

## 💡 Kullanım Örnekleri

### 1. Etkileşimli sohbet (REPL)
```bash
./bin/nexus chat
```
Ya da doğrudan:
```bash
./bin/nexus
```
Gerçek zamanlı R1 akıl yürütme akışı:
```text
nexus ❯ Gökyüzü neden mavidir?

💭 Akıl yürütme süreci:
Kullanıcı, gökyüzünün renginin fiziksel nedenini soruyor...
Atmosferdeki moleküllerin güneş ışığını Rayleigh saçılımına uğratma mekanizmasını açıklamalıyım...

🤖 DeepSeek (deepseek-reasoner):
Gökyüzünün mavi görünmesinin temel nedeni Rayleigh saçılımı adı verilen bir olgudur...
```

### 2. Tek seferlik istem
```bash
./bin/nexus "Genel görelilik hakkında 3 önemli bilgi ver"
```

### 3. Unix pipe ile kod analizi
```bash
cat main.go | ./bin/nexus "Bu kodu güvenlik açıkları ve performans darboğazları açısından incele"
```

### 4. Yerel API Proxy sunucusunu çalıştırma
```bash
./bin/nexus serve --port 9655
```
Artık tüm araçları (Cursor, Open WebUI, Claude Code) `http://localhost:9655/v1` adresine yönlendirebilirsiniz!

**OpenAI biçiminde örnek cURL isteği:**
```bash
curl http://localhost:9655/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-reasoner",
    "messages": [{"role": "user", "content": "Merhaba!"}],
    "stream": true
  }'
```

### 5. Yerel Ollama API benzetimi 🦙
Nexus, Ollama API belirtimini uygular. Open WebUI, Dify, AnythingLLM, Cursor, LlamaIndex ve resmî Ollama kütüphaneleri doğrudan `http://localhost:9655` adresine bağlanabilir:

```bash
# 1. Kullanılabilir modelleri getir (Ollama biçimi)
curl http://localhost:9655/api/tags

# 2. Ollama sohbet isteği (NDJSON akışı)
curl http://localhost:9655/api/chat -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "Merhaba!"}],
  "stream": true
}'
```

**Resmî Python Ollama SDK’sını kullanma:**
```python
import ollama

client = ollama.Client(host="http://localhost:9655")
resp = client.chat(model="deepseek-reasoner", messages=[
    {"role": "user", "content": "Go ile basit bir HTTP sunucusu yaz"}
])
print(resp["message"]["content"])
```

### 6. Yapılandırılmış çıktılar ve JSON Schema 🎯
Katı JSON çıktısı alın veya çıktının belirtilen bir şemaya uymasını sağlayın:

```bash
# 1. CLI üzerinden doğrudan JSON (jq ile pipe üzerinden kullanılabilir)
./bin/nexus "En yaygın 3 programlama dili ve başlıca kullanım alanları" --format json | jq .

# 2. Ollama API üzerinden:
curl http://localhost:9655/api/chat -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "JSON biçiminde 3 meyve listele"}],
  "format": "json",
  "stream": false
}'

# 3. OpenAI API üzerinden (response_format):
curl http://localhost:9655/v1/chat/completions -d '{
  "model": "deepseek-reasoner",
  "messages": [{"role": "user", "content": "Cihaz özelliklerini JSON olarak döndür"}],
  "response_format": {"type": "json_object"}
}'
```

### 7. Evrensel tümleştirmeler (Launcher) 🚀
Kodlama araçlarını ve ajanları tek komutla başlatın — Nexus proxy sunucusunu arka planda otomatik başlatır:

```bash
# 1. Claude Code CLI (DeepSeek R1 / V3 destekli)
./bin/nexus launch claude
./bin/nexus launch claude -p "Bu Go kodunu yeniden düzenle"

# 2. Resmî DeepSeek Harness CLI (@deepseek-ai/dsh)
./bin/nexus launch dsh

# 3. OpenAI Codex CLI (responses wire API ile otomatik yapılandırılır)
./bin/nexus launch codex

# 4. OpenCode CLI (yapılandırma otomatik eklenir)
./bin/nexus launch opencode

# 5. Block Goose AI Agent (OpenAI provider üzerinden)
./bin/nexus launch goose

# 6. Aider Pair Programming CLI (DeepSeek R1 ile)
./bin/nexus launch aider

# 7. ChatGPT Desktop (macOS) ve yerel köprü
./bin/nexus launch chatgpt

# 8. Çoklu Hesap Havuzu İzleyicisi
./bin/nexus launch pool

# 9. DeepSeek Agentic Eval ve Benchmark Harness
./bin/nexus launch harness
```

### 8. Çoklu hesap havuzu 👥
Round-Robin sıralamasıyla birden fazla DeepSeek hesabı arasında geçiş yaparak istek sınırı engellerini (HTTP 429) aşın:

```bash
# Yeni hesap ekle
./bin/nexus auth add acc2 --chrome

# Hesapları listele ve durumlarını denetle
./bin/nexus auth list

# Etkin hesabı değiştir
./bin/nexus auth switch acc2

# Havuzdaki tüm hesapları doğrula
./bin/nexus auth test
```

### 9. Sistem tanılaması (Doctor)
```bash
./bin/nexus doctor
```

---

## 📁 Proje Yapısı

```text
nexus-cli/
├── bin/
│   └── nexus                     # Derlenmiş çalıştırılabilir dosya
├── pkg/
│   ├── deepseek/
│   │   ├── pow.go                # Saf Go ile paralel DeepSeekHashV1 Keccak PoW çözücüsü
│   │   ├── pow_test.go           # PoW testleri ve benchmark’lar
│   │   ├── stream.go             # SSE JSON patch ve OpenAI akış ayrıştırıcısı
│   │   ├── stream_test.go        # Akış testleri
│   │   ├── auth.go               # Belirteç, oturum ve profil yöneticisi
│   │   ├── chrome_auth.go        # Chrome CDP kimlik doğrulamasını otomatikleştirir
│   │   ├── client.go             # DeepSeek Web API istemcisi (Chat, Sessions, Auto-retry)
│   │   └── types.go              # Model yapılandırmaları ve tür tanımları
│   ├── server/
│   │   ├── server.go             # Yerel HTTP proxy sunucusu (Routing, CORS, Auth)
│   │   ├── server_test.go        # Sunucu uç noktası birim testleri
│   │   ├── openai.go             # OpenAI Chat Completions API adaptörü
│   │   ├── anthropic.go          # Anthropic Messages API adaptörü (Claude Code için)
│   │   └── session.go            # Agent oturumlarının yalıtımı ve temizlenmesi
│   ├── launcher/
│   │   └── launcher.go           # Claude Code, Codex, Goose, Aider vb. için başlatıcılar
│   └── cli/
│       ├── terminal.go           # ANSI biçimlendirme, banner’lar ve yükleme göstergeleri
│       ├── repl.go               # Etkileşimli terminal sohbeti ve slash komutları
│       ├── runner.go             # Tek seferlik istem ve Unix pipe çalıştırıcısı
│       └── doctor.go             # Uçtan uca sistem tanılaması
├── main.go                       # Uygulamanın giriş noktası
├── go.mod                        # Go module tanımı
├── README.md                     # Ana belgeler (İngilizce)
├── README.tr.md                  # Türkçe belgeler
├── README.ru.md                  # Rusça belgeler (Русский)
└── README.uz.md                  # Özbekçe belgeler (O'zbekcha)
```

---

## 📄 Lisans

MIT Lisansı altında dağıtılır.
