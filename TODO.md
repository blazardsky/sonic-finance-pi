# Roadmap

## v1.2
- [x] mettere un filtro "recenti / anno" anche in entrate (come in spese)
- [x] vista "modal" e tabella semplificata (senza colonna motivo, dettagli e ricevuto da) anche per le entrate (come in spese)
- [x] ordine clienti per entrate
- [x] color coding per le categorie e visualizzazione raggruppata (per ambito di applicazione)

## v1.2.1
- [x] Cambio sistema reminder per "fatture questo mese" in home/dashboard: da clienti con fatture negli ultimi 3 mesi a clienti con un contratto in corso e non completamente pagato
- [x] Allineare l'estetica dei pulsanti delle card e dei selettori date
- [x] sistemare modale per la cancellazione delle sotto-categorie (testo e pulsanti escondo dal modal)
- [x] Aggiungere possibilità di modificare i contratti già creati
- [x] Togliere la scritta rossa "da fatturare questo mese" per i contratti dove l'ammontare è > al pattuito nel contratto
- [x] aggiungere l'ammontare nella preview della dashboard "clienti questo mese" per ogni fattura, accanto al nome del cliente

## v1.2.2
- [x] Aggregare le spese nel tracker per categoria prima che per singolo prodotto
- [x] aggiungere skeletons per caricamento chart (in particolare alla homepage)
- [x] mettere il tooltip che appare nelle chart come z-index superiore alla legenda
- [x] card su report annuale da rendere più visibili (stile dashboard)
- [x] Fix combobox (es. selezione cliente) non cliccabile su mobile
- [x] Tenere traccia delle quantità di quote possedute per investimento (opzionale, manuale): prezzo attuale e correzione quote a mano nella pagina Titoli, valore/guadagno-perdita calcolati e mostrati in un dettaglio separato

## v1.2.3
- [x] "Fatture da fare" in dashboard: mostrare la quota del mese invece del totale rimanente del contratto, ed escludere i clienti già fatturati questo mese (anche se non ancora pagati)

## v1.2.4
- [ ] Fix: aprire in automatico la sidebar da mobile su "modifica"
- [ ] Pulsanti "pagina precedente" e "pagina successiva" da trasformare in frecce su mobile
- [ ] pagina successiva / pagina precedente ripetuti anche in fondo alla lista
- [ ] usare badge con color coding per le categorie nelle liste spese / entrate
- [ ] nelle entrate da freelance, aggiungere due campi opzionali: numero fattura (che potrebbe anche essere calcolato in automatico, diviso per persona – es. nicco/sofi) e se parte di un contratto aggiungere un campo "importo extra" es per rimborso spesa, mora, indennizzo... che conta ai fini dell'importo incassato ma non ai fini del totale del contratto
- [ ] FIX UI: nel dettaglio spesa, impostare un limite di larghezza max (che da mobile sia inferiore al 100% - padding per evitare che le note siano su una sola riga)
- [ ] Miglioramento UX su mobile: chiusura automatica una volta inserito un record della action bar
- [ ] Aggiunta funzionalità spese: nel prezzo delle voci aggiungere la possibilità di scrivere il valore per costo e quantità come moltiplicazione o sottrazione o somma "x*y" / "x-y", "x+y" e calcolare automaticamente "z" in modo da inserire in una voce due prodotti identici o degli sconti / aggiunte con facilità senza fare il calcolo a mano
- [ ] FIX UI: il combobox non fa cliccare sul selettore del testo consigliato

## v1.2.5
- [ ] Aggiungere il tema Alta visibilità / Alto contrasto
- [ ] card su report annuale da spostare su anno
- [ ] Aggiungere nelle ricorrenti anche le entrate, per esempio per uno stipendio fisso

## v1.3
- [ ] usare tabelle reali (datatable tanstack table) con filtri per colonna

## v1.3.1
- [ ] aggiungere delle chart per le sotto-categorie in report mensile e annuale
- [ ] approfondire la reportistica (e l'esportazione: sankey, viste riassuntive, ...)

## v1.4
- [ ] Aggiungere un mcp per interagire con l'app direttamente da claude code o qualche altra AI

## v1.5
- [ ] Aggiungere un sistema per caricare le spese a partire dalla foto di uno scontrino tramite OCR e probabilmente una AI

## v1.6
- [ ] Aggiungere un sistema per programmare delle spese, capire i momenti migliori per fare acquisti (in base a spese fatte, spese ricorrenti, entrate, stato dei contratti, andamento degli investimenti ecc)

## v1.7
- [ ] drag & drop di widget nella home con possibilità di salvare più profili di visualizzazione
- [ ] aggiungere config (con effetto solo locale, indexeddb) per piccole personalizzazioni a livello di tema (es grandezza del font, larghezza contenuti cliccabili, chiusura automatica una volta inserite i vari record)
