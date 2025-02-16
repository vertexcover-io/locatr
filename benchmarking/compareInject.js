function compareLocators(locators1, locators2) {
    function getElements(locator) {
        try {
            if (locator.startsWith('/')) {
                let result = document.evaluate(locator, document, null, XPathResult.ORDERED_NODE_SNAPSHOT_TYPE, null);
                let elements = [];
                for (let i = 0; i < result.snapshotLength; i++) {
                    elements.push(result.snapshotItem(i));
                }
                return elements;
            } else {
                return Array.from(document.querySelectorAll(locator));
            }
        } catch (e) {
            console.error("❌ Error processing locator:", locator, e);
            return [];
        }
    }

    function getElementCenter(el) {
        let rect = el.getBoundingClientRect();
        return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
    }

    function getDistance(el1, el2) {
        let c1 = getElementCenter(el1);
        let c2 = getElementCenter(el2);
        return Math.sqrt((c1.x - c2.x) ** 2 + (c1.y - c2.y) ** 2);
    }

    function areSameElement(el1, el2) {
        return el1 === el2;
    }

    function areCloseInHierarchy(el1, el2, maxDepth = 2) {
        let p1 = el1, p2 = el2;
        for (let i = 0; i <= maxDepth; i++) {
            if (p1 === el2 || p2 === el1) return true; // Direct parent-child match
            if (p1.parentElement) p1 = p1.parentElement;
            if (p2.parentElement) p2 = p2.parentElement;
        }
        return false;
    }

    for (let loc1 of locators1) {
        const elements1 = getElements(loc1);
        for (let loc2 of locators2) {
            const elements2 = getElements(loc2);
            for (let el1 of elements1) {
                for (let el2 of elements2) {
                    if (areSameElement(el1, el2)) {
                        console.log(`✅ Exact Match: ${loc1} ↔ ${loc2} (Distance: 0)`);
                        return 0;
                    } else if (areCloseInHierarchy(el1, el2)) {
                        let distance = getDistance(el1, el2);
                        console.log(`🔍 Close Match: ${loc1} ↔ ${loc2} (Distance: ${distance})`);
                        return distance;
                    }
                }
            }
        }
    }

    console.log("❌ No match found.");
    return null;
}
