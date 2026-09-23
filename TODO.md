# Miglioramenti vari

## fatto
- [x] mettere un filtro "recenti / anno" anche in entrate (come in spese)
- [x] vista "modal" e tabella semplificata (senza colonna motivo, dettagli e ricevuto da) anche per le entrate (come in spese)
- [x] ordine clienti per entrate
- [x] color coding per le categorie e visualizzazione raggruppata (per ambito di applicazione)
- [x] Cambio sistema reminder per "fatture questo mese" in home/dashboard: da clienti con fatture negli ultimi 3 mesi a clienti con un contratto in corso e non completamente pagato
- [x] Allineare l'estetica dei pulsanti delle card e dei selettori date
- [x] sistemare modale per la cancellazione delle sotto-categorie (testo e pulsanti escondo dal modal)
- [x] Aggiungere possibilità di modificare i contratti già creati
- [x] Togliere la scritta rossa "da fatturare questo mese" per i contratti dove l'ammontare è > al pattuito nel contratto
- [x] aggiungere l'ammontare nella preview della dashboard "clienti questo mese" per ogni fattura, accanto al nome del cliente
- [x] Aggregare le spese nel tracker per categoria prima che per singolo prodotto
- [x] aggiungere skeletons per caricamento chart (in particolare alla homepage)
- [x] mettere il tooltip che appare nelle chart come z-index superiore alla legenda
- [x] card su report annuale da rendere più visibili (stile dashboard)
- [x] Fix combobox (es. selezione cliente) non cliccabile su mobile
- [x] Tenere traccia delle quantità di quote possedute per investimento (opzionale, manuale): prezzo attuale e correzione quote a mano nella pagina Titoli, valore/guadagno-perdita calcolati e mostrati in un dettaglio separato
- [x] "Fatture da fare" in dashboard: mostrare la quota del mese invece del totale rimanente del contratto, ed escludere i clienti già fatturati questo mese (anche se non ancora pagati)
- [x] Fix: aprire in automatico la sidebar da mobile su "modifica"
- [x] Pulsanti "pagina precedente" e "pagina successiva" da trasformare in frecce su mobile
- [x] pagina successiva / pagina precedente ripetuti anche in fondo alla lista
- [x] usare badge con color coding per le categorie nelle liste spese / entrate
- [x] nelle entrate da freelance, aggiungere due campi opzionali: numero fattura (che potrebbe anche essere calcolato in automatico, diviso per persona – es. nicco/sofi) e se parte di un contratto aggiungere un campo "importo extra" es per rimborso spesa, mora, indennizzo... che conta ai fini dell'importo incassato ma non ai fini del totale del contratto
- [x] FIX UI: nel dettaglio spesa, impostare un limite di larghezza max (che da mobile sia inferiore al 100% - padding per evitare che le note siano su una sola riga)
- [x] Miglioramento UX su mobile: chiusura automatica una volta inserito un record della action bar
- [x] Aggiunta funzionalità spese: nel prezzo delle voci aggiungere la possibilità di scrivere il valore per costo e quantità come moltiplicazione o sottrazione o somma "x*y" / "x-y", "x+y" e calcolare automaticamente "z" in modo da inserire in una voce due prodotti identici o degli sconti / aggiunte con facilità senza fare il calcolo a mano
- [x] FIX UI: il combobox non fa cliccare sul selettore del testo consigliato
- [x] Aggiungere il tema Alta visibilità / Alto contrasto
- ~~card su report annuale da spostare su anno~~ **RIFIUTATO**
- [x] usare tabelle reali (datatable tanstack table) con filtri per colonna
- [x] DataTable: mostrare 25 righe per pagina di default (prima 10)
- [x] DataTable: nascondere i filtri per colonna di default, con un pulsante per mostrarli/nasconderli (anche su mobile, stesso comportamento del filtro a tendina già esistente)
- [x] Clienti: ordinare i contratti nel pannello espanso dal più recente
- [x] Clienti: nascondere di default i contratti già completati (terminati, fatturati e incassati per intero) dietro un pulsante "mostra completati", invece di lasciarli sempre in vista
- [x] Fix: il salvataggio di una modifica in Investimenti non deve più resettare la tabella a pagina 1 (paginazione ora persistita nell'URL)
- [x] Titoli: aggiungere il prezzo medio d'acquisto per quota, calcolato da quote e costo — modificabile aggiungendo un nuovo acquisto (si somma al totale) o sovrascrivendo quote/prezzo medio esistenti

## da fare
- [ ] approfondire la reportistica (e l'esportazione: sankey, viste riassuntive, ...)
- [x] spostare la card "risparmio iniziale" dalla pagina risparmi alla pagina impostazioni
- [ ] Aggiungere un sistema per caricare le spese a partire dalla foto di uno scontrino tramite OCR e probabilmente una AI
- [x] Aggiungere un sistema per programmare delle spese, capire i momenti migliori per fare acquisti (in base a spese fatte, spese ricorrenti, entrate, stato dei contratti, andamento degli investimenti ecc)
- [ ] aggiungere config (con effetto solo locale, indexeddb) per piccole personalizzazioni a livello di tema (es grandezza del font, larghezza contenuti cliccabili, chiusura automatica una volta inserite i vari record)

# da valutare
- [ ] Aggiungere nelle ricorrenti anche le entrate, per esempio per uno stipendio fisso
- [ ] drag & drop di widget nella home con possibilità di salvare più profili di visualizzazione
- [ ] Aggiungere un mcp per interagire con l'app direttamente da claude code o qualche altra AI
- [ ] aggiungere delle chart per le sotto-categorie in report mensile e annuale
